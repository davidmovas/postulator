package tools

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/runs"
	domaingraph "github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	graphEditName    = "graph_edit"
	opAnchors        = "anchors"
	opAddEdge        = "add_edge"
	opApproveEdge    = "approve_edge"
	opRejectEdge     = "reject_edge"
	opDeleteEdge     = "delete_edge"
	opMove           = "move"
	opRescore        = "rescore"
	opAdoptParents   = "adopt_parents"
	opApplyProposals = "apply_proposals"
)

type graphEditArgs struct {
	Op          string                 `json:"op" enum:"create,update,delete,anchors,add_edge,approve_edge,reject_edge,delete_edge,move,rescore,adopt_parents,apply_proposals" description:"adopt_parents writes the page tree's parents a graph_read adoption previewed; apply_proposals writes chosen proposals"`
	EntityID    string                 `json:"entityId,omitempty" description:"update, delete, anchors, move: the entity; add_edge: the child of a parent edge"`
	Name        *string                `json:"name,omitempty" description:"create, update: subject, 2-4 words, title case"`
	Intent      *string                `json:"intent,omitempty" description:"create, update: the reader's goal, one sentence"`
	Keywords    *[]dto.Keyword         `json:"keywords,omitempty" description:"create, update: the whole keyword list, by volume"`
	Definition  *string                `json:"definition,omitempty" description:"update: what the entity is, a sentence or two"`
	Facts       *[]graph.Fact          `json:"facts,omitempty" description:"update: the whole list of key facts"`
	Notes       *[]graph.Note          `json:"notes,omitempty" description:"update: the whole list of notes"`
	Anchors     []graph.Anchor         `json:"anchors,omitempty" description:"create, anchors: the whole list of link texts; anchors without one clears them"`
	ParentID    string                 `json:"parentId,omitempty" description:"create, move: the parent entity; create omits it for a root"`
	Entities    []graph.EntityInput    `json:"entities,omitempty" description:"create: a tree of entities in one write, parents first, in place of name"`
	ToEntityID  string                 `json:"toEntityId,omitempty" description:"add_edge: the parent of a parent edge, or the other entity"`
	Kind        string                 `json:"kind,omitempty" enum:"parent,related" description:"add_edge: parent builds the tree, no cycles; related links siblings"`
	Weight      float64                `json:"weight,omitempty" minimum:"0" maximum:"1" description:"add_edge: closeness; a parent edge takes 1"`
	Status      string                 `json:"status,omitempty" enum:"approved,proposed,rejected" description:"add_edge: default approved; proposed leaves it to the person"`
	Reason      string                 `json:"reason,omitempty" description:"add_edge: why they belong together, one sentence"`
	EdgeID      string                 `json:"edgeId,omitempty" description:"approve_edge, reject_edge, delete_edge: the edge"`
	KeepBoth    bool                   `json:"keepBoth,omitempty" description:"move: keep the old parent too"`
	Under       string                 `json:"under,omitempty" description:"adopt_parents: the subtree's page, as previewed"`
	EntityIDs   []string               `json:"entityIds,omitempty" description:"adopt_parents: the entities, as previewed"`
	Hash        string                 `json:"hash,omitempty" description:"adopt_parents: the preview's hash"`
	Proposals   []graph.ProposedEntity `json:"proposals,omitempty" description:"apply_proposals: the proposals as previewed, minus the declined ones"`
	ProposalsOf string                 `json:"proposalsOf,omitempty" enum:"model,site" description:"apply_proposals: the preview's source"`
}

func graphEdit(deps Deps) Tool {
	tool := newSiteTool(Def{
		Name: graphEditName,
		Description: "Change the entity graph: add entities one or a tree at a time, change, delete or re-anchor one, " +
			"add, approve, reject or delete an edge, move an entity, rescore, adopt the page tree's parents, write proposals.",
		Risk: RiskDangerous,
	}, func(ctx context.Context, b Binding, in graphEditArgs) (any, error) {
		return deps.editGraph(ctx, b, &in)
	})
	tool = confirming(tool, func(ctx context.Context, b Binding, in graphEditArgs) (string, error) {
		switch in.Op {
		case opApplyProposals:
			return proposalsSummary(in.proposals(b.SiteID))
		case opAdoptParents:
			return deps.adoptionSummary(ctx, b, &in)
		default:
			return "", nil
		}
	})
	tool = describing(tool, func(ctx context.Context, b Binding, in graphEditArgs) (string, error) {
		return deps.graphChange(ctx, &in)
	})
	tool.Adds = func(args json.RawMessage) []runs.AssumedEdge {
		var in graphEditArgs
		if json.Unmarshal(args, &in) != nil || in.Op != opAddEdge || in.Status != "" && in.Status != string(domaingraph.StatusApproved) {
			return nil
		}
		return []runs.AssumedEdge{{FromEntityID: in.EntityID, ToEntityID: in.ToEntityID}}
	}
	return riskedBy(tool, map[string]Risk{
		opCreate: RiskWrite, opUpdate: RiskWrite, opAnchors: RiskWrite, opAddEdge: RiskWrite, opApproveEdge: RiskWrite,
		opRejectEdge: RiskWrite, opMove: RiskWrite, opRescore: RiskWrite, opAdoptParents: RiskWrite, opApplyProposals: RiskWrite,
	})
}

func (in *graphEditArgs) proposals(siteID string) graph.ApplyProposalsRequest {
	return graph.ApplyProposalsRequest{SiteID: siteID, Entities: in.Proposals, Source: graph.ProposalSource(in.ProposalsOf)}
}

func (deps Deps) editGraph(ctx context.Context, b Binding, in *graphEditArgs) (any, error) {
	switch in.Op {
	case opCreate:
		if len(in.Entities) > 0 {
			return deps.Graph.CreateEntities(ctx, graph.CreateEntitiesRequest{SiteID: b.SiteID, Entities: in.Entities})
		}
		return deps.Graph.CreateEntity(ctx, graph.CreateEntityRequest{
			SiteID: b.SiteID, Name: valueOf(in.Name), Intent: valueOf(in.Intent), Keywords: valueOf(in.Keywords),
			Anchors: in.Anchors, ParentID: in.ParentID,
		})
	case opUpdate:
		return deps.Graph.UpdateEntity(ctx, graph.UpdateEntityRequest{
			ID: in.EntityID, Name: in.Name, Intent: in.Intent, Keywords: in.Keywords, Definition: in.Definition,
			Facts: in.Facts, Notes: in.Notes,
		})
	case opDelete:
		return deps.Graph.DeleteEntity(ctx, graph.DeleteEntityRequest{ID: in.EntityID})
	case opAnchors:
		return deps.Graph.SetAnchors(ctx, graph.SetAnchorsRequest{EntityID: in.EntityID, Anchors: in.Anchors})
	case opAddEdge:
		return deps.Graph.AddEdge(ctx, graph.AddEdgeRequest{
			SiteID: b.SiteID, FromEntityID: in.EntityID, ToEntityID: in.ToEntityID, Kind: in.Kind, Weight: in.Weight,
			Status: in.Status, Reason: in.Reason,
		})
	case opApproveEdge:
		return deps.Graph.ApproveEdge(ctx, graph.ApproveEdgeRequest{ID: in.EdgeID})
	case opRejectEdge:
		return deps.Graph.RejectEdge(ctx, graph.RejectEdgeRequest{ID: in.EdgeID})
	case opDeleteEdge:
		return deps.Graph.DeleteEdge(ctx, graph.DeleteEdgeRequest{ID: in.EdgeID})
	case opMove:
		return deps.Graph.MoveEntity(ctx, graph.MoveEntityRequest{EntityID: in.EntityID, NewParentID: in.ParentID, KeepBoth: in.KeepBoth})
	case opRescore:
		return deps.Graph.RecomputeScores(ctx, graph.RecomputeScoresRequest{SiteID: b.SiteID})
	case opAdoptParents:
		return deps.Graph.AdoptParents(ctx, graph.AdoptParentsRequest{
			SiteID: b.SiteID, Under: in.Under, EntityIDs: in.EntityIDs, Hash: in.Hash, Apply: true,
		})
	case opApplyProposals:
		return deps.Graph.ApplyProposals(ctx, in.proposals(b.SiteID))
	default:
		return nil, errors.New(errors.Invalid, "graph_edit takes the op create, update, delete, anchors, add_edge, approve_edge, "+
			"reject_edge, delete_edge, move, rescore, adopt_parents or apply_proposals").WithDetail("field", "op")
	}
}

func (deps Deps) adoptionSummary(ctx context.Context, b Binding, in *graphEditArgs) (string, error) {
	if strings.TrimSpace(in.Hash) == "" {
		return "", errors.New(errors.Invalid, "adopt_parents writes a preview: give the hash graph_read adoption answered").
			WithDetail("field", "hash")
	}
	preview, err := deps.Graph.AdoptParents(ctx, graph.AdoptParentsRequest{
		SiteID: b.SiteID, Under: in.Under, EntityIDs: in.EntityIDs, Limit: proposalsNamed,
	})
	if err != nil {
		return "", err
	}
	if preview.Hash != in.Hash {
		return "", errors.New(errors.Conflict, "the page tree or the graph moved since the preview; read the adoption again").
			WithDetail("hash", preview.Hash)
	}
	named := make([]string, 0, proposalsNamed)
	for i := range preview.Steps.Items {
		step := &preview.Steps.Items[i]
		if step.Skip != "" || len(named) == proposalsNamed {
			continue
		}
		to := "no parent"
		if step.To != nil {
			to = step.To.Name
		}
		named = append(named, step.Name+" under "+to)
	}
	said := "Give " + entities(preview.Adopting) + " the parent the page tree shows"
	if len(named) > 0 {
		said += ": " + strings.Join(named, ", ")
		if more := preview.Adopting - len(named); more > 0 {
			said += " and " + strconv.Itoa(more) + " more"
		}
	}
	if preview.Skipped > 0 {
		said += "; " + entities(preview.Skipped) + " skipped, each for a loop or a name already under that parent"
	}
	return said, nil
}

func entities(count int) string {
	if count == 1 {
		return "1 entity"
	}
	return strconv.Itoa(count) + " entities"
}

func (deps Deps) graphChange(ctx context.Context, in *graphEditArgs) (string, error) {
	switch in.Op {
	case opCreate:
		return createdSummary(in), nil
	case opRescore:
		return "Recompute the authority score of every entity of the site", nil
	case opApproveEdge, opRejectEdge, opDeleteEdge:
		return deps.edgeChange(ctx, in)
	}
	name, err := deps.entityName(ctx, in.EntityID)
	if err != nil {
		return "", err
	}
	switch in.Op {
	case opUpdate:
		return "Change the " + strings.Join(in.changed(), ", ") + " of the entity " + name, nil
	case opDelete:
		return "Delete the entity " + name + " and the edges that touch it", nil
	case opAnchors:
		if len(in.Anchors) == 0 {
			return "Clear every anchor text of " + name, nil
		}
		texts := make([]string, 0, len(in.Anchors))
		for _, anchor := range in.Anchors {
			texts = append(texts, quoted(anchor.Text))
		}
		return "Let pages link to " + name + " with " + strings.Join(texts, ", "), nil
	case opAddEdge:
		other, otherErr := deps.entityName(ctx, in.ToEntityID)
		if otherErr != nil {
			return "", otherErr
		}
		said, saying := "Relate "+name+" to "+other, "relating "+name+" to "+other
		if in.Kind == string(domaingraph.EdgeParent) {
			said, saying = "Put "+name+" under "+other, "putting "+name+" under "+other
		}
		switch domaingraph.EdgeStatus(in.Status) {
		case domaingraph.StatusProposed:
			return "Propose " + saying + ", the edge left for a person to approve", nil
		case domaingraph.StatusRejected:
			return "Record " + saying + " as rejected", nil
		default:
			return said + " in the graph", nil
		}
	default:
		parent, parentErr := deps.entityName(ctx, in.ParentID)
		if parentErr != nil {
			return "", parentErr
		}
		if in.KeepBoth {
			return "Give " + name + " a second parent, " + parent, nil
		}
		return "Move " + name + " under " + parent, nil
	}
}

func createdSummary(in *graphEditArgs) string {
	if len(in.Entities) == 0 {
		return "Add the entity " + valueOf(in.Name) + " to the graph"
	}
	named := make([]string, 0, proposalsNamed)
	for i := range in.Entities[:min(len(in.Entities), proposalsNamed)] {
		named = append(named, in.Entities[i].Name)
	}
	said := "Add " + entities(len(in.Entities)) + " to the graph: " + strings.Join(named, ", ")
	if more := len(in.Entities) - len(named); more > 0 {
		said += " and " + strconv.Itoa(more) + " more"
	}
	return said
}

func (deps Deps) edgeChange(ctx context.Context, in *graphEditArgs) (string, error) {
	held, err := deps.Graph.GetEdge(ctx, graph.GetEdgeRequest{ID: in.EdgeID})
	if err != nil {
		return "", err
	}
	from, err := deps.entityName(ctx, held.Edge.FromEntityID)
	if err != nil {
		return "", err
	}
	to, err := deps.entityName(ctx, held.Edge.ToEntityID)
	if err != nil {
		return "", err
	}
	edge := "the related edge between " + from + " and " + to
	if held.Edge.Kind == "parent" {
		edge = "the parent edge putting " + from + " under " + to
	}
	switch in.Op {
	case opApproveEdge:
		return "Approve " + edge + " so the link rules use it", nil
	case opRejectEdge:
		return "Reject " + edge, nil
	default:
		return "Delete " + edge, nil
	}
}

func (in *graphEditArgs) changed() []string {
	named := []struct {
		set  bool
		name string
	}{
		{in.Name != nil, "name"}, {in.Intent != nil, "intent"}, {in.Keywords != nil, "keywords"},
		{in.Definition != nil, "definition"}, {in.Facts != nil, "facts"}, {in.Notes != nil, "notes"},
	}
	out := make([]string, 0, len(named))
	for _, held := range named {
		if held.set {
			out = append(out, held.name)
		}
	}
	if len(out) == 0 {
		return []string{"nothing"}
	}
	return out
}
