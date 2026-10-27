package graph

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/finding"
	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	adoptUnit         = 1000
	adoptDefaultLimit = 50
	adoptMaxLimit     = 200
)

type AdoptParentsRequest struct {
	SiteID    string        `json:"siteId"`
	Under     string        `json:"under,omitempty" description:"A page id: adopt only for the entities of its subtree; omit for the whole site"`
	EntityIDs []string      `json:"entityIds,omitempty" description:"Only these entities"`
	Hash      string        `json:"hash,omitempty" description:"apply: the hash of the preview the person saw"`
	Cursor    paging.Cursor `json:"cursor,omitempty" description:"preview: nextCursor of the last call"`
	Limit     int           `json:"limit,omitempty" description:"preview: default 50, at most 200"`
	Apply     bool          `json:"apply,omitempty" description:"Write the preview whose hash is given"`
}

type EntityName struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type AdoptionView struct {
	To       *EntityName  `json:"to"`
	EntityID string       `json:"entityId"`
	Name     string       `json:"name"`
	Path     string       `json:"path"`
	ToPath   string       `json:"toPath,omitempty"`
	Skip     string       `json:"skip,omitempty"`
	From     []EntityName `json:"from"`
}

type AdoptParentsResponse struct {
	Steps    paging.List[AdoptionView] `json:"steps"`
	Hash     string                    `json:"hash"`
	Adopting int                       `json:"adopting"`
	Skipped  int                       `json:"skipped"`
	Applied  bool                      `json:"applied"`
}

func (s *Service) AdoptParents(ctx context.Context, req AdoptParentsRequest) (AdoptParentsResponse, error) {
	siteID := strings.TrimSpace(req.SiteID)
	if err := requireSite(siteID); err != nil {
		return AdoptParentsResponse{}, err
	}
	if s.maps == nil {
		return AdoptParentsResponse{}, errors.New(errors.Invalid, "no site model is wired for the page tree's parents")
	}
	tree, err := s.maps.Map(ctx, siteID)
	if err != nil {
		return AdoptParentsResponse{}, err
	}
	var pageIDs []string
	if req.Under != "" {
		if pageIDs = tree.Under(req.Under); len(pageIDs) == 0 {
			return AdoptParentsResponse{}, errors.New(errors.NotFound, "the site's map holds no such page").WithDetail("under", req.Under)
		}
	}
	plan := tree.Adoptions(pageIDs, req.EntityIDs)
	response := AdoptParentsResponse{Hash: adoptionHash(plan)}
	adopting := make([]contextpack.Adoption, 0, len(plan))
	for i := range plan {
		if plan[i].Skip != "" {
			response.Skipped++
			continue
		}
		adopting = append(adopting, plan[i])
	}
	response.Adopting = len(adopting)

	if req.Apply {
		if req.Hash != response.Hash {
			return AdoptParentsResponse{}, errors.New(errors.Conflict, "the page tree or the graph moved since the preview; preview the adoption again").
				WithDetail("hash", response.Hash)
		}
		if adoptErr := s.adoptParents(ctx, siteID, adopting); adoptErr != nil {
			return AdoptParentsResponse{}, adoptErr
		}
		response.Applied = true
	}
	views := make([]AdoptionView, 0, len(plan))
	for i := range plan {
		views = append(views, adoptionView(tree, plan[i]))
	}
	slices.SortFunc(views, func(a, b AdoptionView) int {
		return cmp.Or(cmp.Compare(a.Path, b.Path), cmp.Compare(a.EntityID, b.EntityID))
	})
	keyset := paging.Keyset[AdoptionView]{
		IDColumn: "entity_id",
		ID:       func(view AdoptionView) string { return view.EntityID },
		Keys:     []paging.SortKey[AdoptionView]{paging.TextKey("path", "path", func(view AdoptionView) any { return view.Path })},
	}
	limit := adoptDefaultLimit
	if req.Limit > 0 {
		limit = min(req.Limit, adoptMaxLimit)
	}
	if response.Steps, err = keyset.Page(views, paging.Request{After: req.Cursor, Limit: limit}); err != nil {
		return AdoptParentsResponse{}, err
	}
	return response, nil
}

func (s *Service) adoptParents(ctx context.Context, siteID string, steps []contextpack.Adoption) error {
	for start := 0; start < len(steps); start += adoptUnit {
		chunk := steps[start:min(start+adoptUnit, len(steps))]
		err := s.uow.Do(ctx, func(c context.Context) error {
			edges, err := s.edges.ListBySite(c, siteID)
			if err != nil {
				return err
			}
			for i := range chunk {
				if adoptErr := s.adoptOne(c, siteID, edges, chunk[i]); adoptErr != nil {
					return adoptErr
				}
			}
			return s.settleScopes(c, siteID)
		})
		if err != nil {
			return err
		}
	}
	resolvedBy := finding.ResolvedByPerson
	if actor, ok := kctx.ActorFrom(ctx); ok && actor == kctx.ActorAgent {
		resolvedBy = finding.ResolvedByAgent
	}
	now := s.now()
	for i := range steps {
		if err := s.findings.Clear(ctx, siteID, finding.EntityParentDiffers, finding.SubjectEntityParent(steps[i].EntityID),
			resolvedBy, finding.ResolutionParentAdopted, now); err != nil {
			return err
		}
	}
	if len(steps) == 0 {
		return nil
	}
	if err := s.parentsMoved(ctx, siteID); err != nil {
		return err
	}
	return s.changed(siteID)
}

func (s *Service) adoptOne(ctx context.Context, siteID string, edges []graphdomain.Edge, step contextpack.Adoption) error {
	held, replaced := parentEdges(edges, step.EntityID, step.To)
	for i := range replaced {
		if err := s.edges.Delete(ctx, replaced[i].ID); err != nil {
			return err
		}
	}
	if step.To == "" || held != nil {
		return nil
	}
	for i := range edges {
		if edges[i].Kind == graphdomain.EdgeParent && edges[i].FromEntityID == step.EntityID && edges[i].ToEntityID == step.To {
			return s.edges.SetStatus(ctx, edges[i].ID, graphdomain.StatusApproved)
		}
	}
	edge, err := graphdomain.NewEdge(graphdomain.Edge{
		ID: id.New(), SiteID: siteID, FromEntityID: step.EntityID, ToEntityID: step.To, Kind: graphdomain.EdgeParent,
		Weight: 1, Source: sourceOfActor(ctx), Status: graphdomain.StatusApproved, CreatedAt: s.now(),
		Reason: "the page tree puts " + step.Path + " under " + step.ToPath,
	})
	if err != nil {
		return err
	}
	return s.edges.Insert(ctx, edge)
}

func adoptionView(tree contextpack.Map, step contextpack.Adoption) AdoptionView {
	view := AdoptionView{
		EntityID: step.EntityID, Name: step.Name, Path: step.Path, ToPath: step.ToPath, Skip: string(step.Skip),
		From: make([]EntityName, 0, len(step.From)),
	}
	subject := tree.SubjectOf(contextpack.ParentDisagreement{EntityID: step.EntityID, GraphParents: step.From,
		Tree: contextpack.TreeParent{EntityID: step.To}})
	for i, parentID := range step.From {
		view.From = append(view.From, EntityName{ID: parentID, Name: subject.Graph[i]})
	}
	if step.To != "" {
		view.To = &EntityName{ID: step.To, Name: subject.Tree}
	}
	return view
}

func adoptionHash(plan []contextpack.Adoption) string {
	sum := sha256.New()
	for i := range plan {
		from := slices.Clone(plan[i].From)
		slices.Sort(from)
		sum.Write([]byte(plan[i].EntityID + "|" + strings.Join(from, ",") + "|" + plan[i].To + "|" + string(plan[i].Skip) + "\n"))
	}
	return hex.EncodeToString(sum.Sum(nil))
}
