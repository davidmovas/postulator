package steps

import (
	"context"
	"net/url"
	"strconv"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

const (
	NameRelinkNeighbors = string(run.StepRelinkNeighbors)

	OutcomeLinked    = "linked"
	OutcomeUnchanged = "unchanged"
	OutcomeConflict  = "conflict"
	OutcomeSkipped   = "skipped"

	CodeRelinkConflict = "relink_conflict"
	CodeRelinkSkipped  = "relink_skipped"
	ClassNeedsHuman    = "needs_human"

	relinkTimeout = 5 * time.Minute
)

type NeighborResult struct {
	PageID  string `json:"pageId"`
	Path    string `json:"path"`
	Outcome string `json:"outcome"`
	Anchor  string `json:"anchor"`
	Detail  string `json:"detail"`
	WPID    int64  `json:"wpId"`
}

type RelinkResult struct {
	Neighbors []NeighborResult  `json:"neighbors"`
	Findings  []content.Finding `json:"findings"`
	Linked    int               `json:"linked"`
	Conflicts int               `json:"conflicts"`
	Skipped   int               `json:"skipped"`
}

func RelinkNeighbors(deps Deps) run.StepDef {
	return run.StepDef{
		Name:     NameRelinkNeighbors,
		Requires: []run.ArtifactKind{run.ArtifactLinkContext, run.ArtifactPublishResult},
		Produces: []run.ArtifactKind{run.ArtifactRelinkResult},
		Retry:    run.RetryPolicy{Max: 2},
		Timeout:  relinkTimeout,
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			lc, err := linkContextOf(sc)
			if err != nil {
				return run.Result{}, err
			}
			entity, err := entityOf(ctx, deps, sc)
			if err != nil {
				return run.Result{}, err
			}
			policy, err := effectivePolicy(ctx, deps, sc)
			if err != nil {
				return run.Result{}, err
			}
			owner, err := deps.Sites.Get(ctx, sc.Run.SiteID)
			if err != nil {
				return run.Result{}, err
			}
			client, err := clientFor(ctx, deps, sc.Run.SiteID)
			if err != nil {
				return run.Result{}, err
			}
			neighborhood, err := aroundThePage(ctx, deps, sc)
			if err != nil {
				return run.Result{}, err
			}

			result := RelinkResult{
				Neighbors: make([]NeighborResult, 0, len(lc.Targets)),
				Findings:  make([]content.Finding, 0),
			}

			for i := range lc.Targets {
				neighbor, ok := neighborhood.index.ByID(lc.Targets[i].PageID)
				switch {
				case !ok, neighbor.WPID == nil, neighbor.ID == sc.Page.ID:
					continue
				case lc.Targets[i].EntityID == entity.ID:
					continue
				}

				work := neighborhood
				work.neighbor = neighbor
				work.site = pagemap.NewSite(owner.BaseURL)
				work.page = sc.Page
				work.base = policy

				outcome, relinkErr := relinkOne(ctx, deps, client, work)
				if relinkErr != nil {
					return run.Result{}, relinkErr
				}
				result.Neighbors = append(result.Neighbors, outcome)

				switch outcome.Outcome {
				case OutcomeLinked:
					result.Linked++
				case OutcomeConflict:
					result.Conflicts++
					result.Findings = append(result.Findings, conflictFinding(outcome))
				case OutcomeSkipped:
					result.Skipped++
					result.Findings = append(result.Findings, skippedFinding(outcome))
				}
			}

			blob, err := encode(result, "relink result")
			if err != nil {
				return run.Result{}, err
			}
			return run.Result{
				Artifacts: []run.Artifact{{Kind: run.ArtifactRelinkResult, Blob: blob}},
				Message:   "relinked " + strconv.Itoa(result.Linked) + " neighbors of " + sc.Page.Path,
			}, nil
		},
	}
}

type neighborWork struct {
	neighbor pagemap.Page
	page     pagemap.Page
	site     pagemap.Site
	index    pagemap.Index
	graph    graph.Graph
	base     template.LinkPolicy
}

func aroundThePage(ctx context.Context, deps Deps, sc *run.StepContext) (neighborWork, error) {
	pages, err := deps.Pages.ListBySite(ctx, sc.Run.SiteID)
	if err != nil {
		return neighborWork{}, err
	}
	entities, err := deps.Entities.ListBySite(ctx, sc.Run.SiteID)
	if err != nil {
		return neighborWork{}, err
	}
	edges, err := deps.Edges.ListBySite(ctx, sc.Run.SiteID)
	if err != nil {
		return neighborWork{}, err
	}
	built, err := graph.New(entities, edges)
	if err != nil {
		return neighborWork{}, err
	}
	return neighborWork{index: pagemap.NewIndex(pages), graph: built}, nil
}

func skippedFinding(outcome NeighborResult) content.Finding {
	return content.Finding{
		Severity: content.SeverityWarn,
		Code:     CodeRelinkSkipped,
		Message:  "the neighbor " + outcome.Path + " was not relinked: " + outcome.Detail,
		Details: map[string]any{
			"pageId": outcome.PageID, "wpId": outcome.WPID, "reason": outcome.Detail,
		},
	}
}

func conflictFinding(outcome NeighborResult) content.Finding {
	return content.Finding{
		Severity: content.SeverityWarn,
		Code:     CodeRelinkConflict,
		Message:  "the neighbor " + outcome.Path + " changed on the site while it was being relinked",
		Details: map[string]any{
			"class": ClassNeedsHuman, "pageId": outcome.PageID, "wpId": outcome.WPID,
		},
	}
}

func neighborPlan(ctx context.Context, deps Deps, in neighborWork) (content.LinkContext, template.LinkPolicy, error) {
	resolved, err := deps.Policies.ResolveForPage(ctx, templates.ResolveForPageRequest{PageID: in.neighbor.ID})
	if err != nil {
		return content.LinkContext{}, template.LinkPolicy{}, err
	}

	policy := in.base
	policy.Rules = templates.EffectiveRules(in.base.Rules, resolved.Spec.LinkRules)

	plan := content.PlanLinks(in.graph, in.index, content.Subject{
		Site: in.site, PageID: in.neighbor.ID, PagePath: in.neighbor.Path, EntityID: *in.neighbor.EntityID,
	}, policy)
	return plan.Context, policy, nil
}

func owedReason(lc content.LinkContext, page pagemap.Page) string {
	if page.EntityID == nil {
		return ReasonNeighborOwesNothing
	}
	for i := range lc.Targets {
		if lc.Targets[i].EntityID == *page.EntityID {
			return ReasonNeighborOwesTheCanonicalPage
		}
	}
	return ReasonNeighborOwesNothing
}

func relinkOne(ctx context.Context, deps Deps, client *wp.Client, in neighborWork) (NeighborResult, error) {
	outcome := NeighborResult{PageID: in.neighbor.ID, Path: in.neighbor.Path, WPID: *in.neighbor.WPID}

	if in.neighbor.EntityID == nil {
		return skip(outcome, ReasonNeighborUnmapped), nil
	}
	lc, policy, err := neighborPlan(ctx, deps, in)
	if errors.IsCode(err, errors.NotFound) {
		return skip(outcome, ReasonNeighborNoTemplate), nil
	}
	if err != nil {
		return NeighborResult{}, err
	}

	target, owed := lc.ByPageID(in.page.ID)
	if !owed {
		return skip(outcome, owedReason(lc, in.page)), nil
	}

	raw, err := client.GetRaw(ctx, *in.neighbor.WPID)
	if err != nil {
		switch {
		case wp.IsPluginMissing(err):
			return skip(outcome, ReasonNoPlugin), nil
		case errors.IsCode(err, errors.NotFound):
			return skip(outcome, ReasonNeighborGone), nil
		default:
			return NeighborResult{}, err
		}
	}

	doc, err := content.Parse(raw.Content)
	if err != nil {
		return skip(outcome, ReasonNeighborUnreadable), nil
	}

	placement := content.InsertTarget(doc, lc, policy, target)
	anchor, inserted := insertedAnchor(placement)
	if !inserted {
		outcome.Outcome = OutcomeUnchanged
		outcome.Detail = firstDetail(placement)
		return outcome, nil
	}

	updated := doc.HTML()
	hash, err := client.PutRaw(ctx, *in.neighbor.WPID, updated, raw.ContentHash)
	if err != nil {
		if errors.IsCode(err, errors.Conflict) {
			outcome.Outcome = OutcomeConflict
			outcome.Detail = err.Error()
			return outcome, nil
		}
		return NeighborResult{}, err
	}

	outcome.Outcome = OutcomeLinked
	outcome.Anchor = anchor
	if adoptErr := adopt(ctx, deps, in.neighbor, in.index, in.site, doc, hash); adoptErr != nil {
		return NeighborResult{}, adoptErr
	}
	return outcome, nil
}

func skip(outcome NeighborResult, reason string) NeighborResult {
	outcome.Outcome = OutcomeSkipped
	outcome.Detail = reason
	return outcome
}

func insertedAnchor(placement content.InsertResult) (anchor string, inserted bool) {
	for i := range placement.Decisions {
		if placement.Decisions[i].Outcome == content.OutcomeInserted {
			return placement.Decisions[i].Anchor, true
		}
	}
	return "", false
}

func firstDetail(placement content.InsertResult) string {
	if len(placement.Decisions) == 0 {
		return ""
	}
	return string(placement.Decisions[0].Outcome) + ": " + placement.Decisions[0].Detail
}

func adopt(ctx context.Context, deps Deps, page pagemap.Page, index pagemap.Index, site pagemap.Site,
	doc *content.Document, hash string) error {
	now := deps.now()
	next := page
	next.ContentHash = hash
	next.Drift = false
	next.LastSyncedAt = &now
	next.UpdatedAt = now

	return persist(ctx, deps, next, observedOn(page, index, site, doc.Links(), now))
}

func observedLinks(page pagemap.Page, index pagemap.Index, host string, found []content.Link,
	at time.Time) []pagemap.PageLink {
	return observedOn(page, index, pagemap.Site{Host: host}, found, at)
}

func observedOn(page pagemap.Page, index pagemap.Index, site pagemap.Site, found []content.Link,
	at time.Time) []pagemap.PageLink {
	out := make([]pagemap.PageLink, 0, len(found))
	for i := range found {
		path, kind := site.Resolve(found[i].Href)
		if kind != pagemap.LinkPath || path == "" {
			continue
		}

		link := pagemap.PageLink{
			ID: id.New(), SiteID: page.SiteID, FromPageID: page.ID,
			ToURL: path, AnchorText: found[i].Anchor, Origin: pagemap.OriginObserved, ObservedAt: at,
		}
		if target, ok := index.ByPath(path); ok {
			link.ToPageID = &target.ID
		}
		out = append(out, link)
	}
	return out
}

func hostOf(baseURL string) string {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	return parsed.Host
}
