package steps

import (
	"context"
	"strconv"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/run"
)

const (
	NameRelinkNeighbors = string(run.StepRelinkNeighbors)

	OutcomeLinked    = "linked"
	OutcomeUnchanged = "unchanged"
	OutcomeConflict  = "conflict"
	OutcomeSkipped   = "skipped"

	CodeRelinkConflict        = "relink_conflict"
	CodeRelinkSkipped         = "relink_skipped"
	CodeRelinkPhraseTemplated = "relink_phrase_templated"
	CodeNeighborLinkMissing   = "neighbor_link_missing"
	ClassNeedsHuman           = "needs_human"

	ReasonNeighborGone       = "the neighbor is no longer on the site"
	ReasonNeighborUnreadable = "the stored content of the neighbor could not be read as HTML"
	ReasonNeighborUnmapped   = "the neighbor is not mapped to an entity, so it has no rules of its own"
	ReasonNeighborIsATerm    = "the neighbor is a product category, whose description the companion plugin does not write"

	ReasonNeighborOwesNothing = "the rules of the neighbor do not ask it to link to this page"

	ReasonNeighborOwesTheCanonicalPage = "the rules of the neighbor ask it to link to the canonical page " +
		"of this entity, not to this page"

	relinkTimeout = 5 * time.Minute
)

type NeighborBefore struct {
	Hash string `json:"hash"`
	HTML string `json:"html"`
}

type NeighborResult struct {
	PageID   string         `json:"pageId"`
	Path     string         `json:"path"`
	Type     string         `json:"type,omitempty"`
	Outcome  string         `json:"outcome"`
	Anchor   string         `json:"anchor"`
	Sentence string         `json:"sentence,omitempty"`
	Detail   string         `json:"detail"`
	Before   NeighborBefore `json:"before"`
	WPID     int64          `json:"wpId"`
	missing  bool
}

type RelinkResult struct {
	Neighbors []NeighborResult  `json:"neighbors"`
	Findings  []content.Finding `json:"findings"`
	Linked    int               `json:"linked"`
	Conflicts int               `json:"conflicts"`
	Skipped   int               `json:"skipped"`
	Missing   int               `json:"missing"`
}

func RelinkNeighbors(deps Deps) run.StepDef {
	return run.StepDef{
		Name:      NameRelinkNeighbors,
		Preflight: pluginPreflight(deps, NameRelinkNeighbors, "cannot read or write the neighbors and stands down"),
		Requires:  []run.ArtifactKind{run.ArtifactLinkContext, run.ArtifactPublishResult},
		Produces:  []run.ArtifactKind{run.ArtifactRelinkResult},
		Retry:     run.RetryPolicy{Max: 2},
		Timeout:   relinkTimeout,
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			if _, err := sc.Artifact(run.ArtifactPublishResult); err != nil {
				return run.Result{}, err
			}
			published, _, err := decodeArtifact[PublishResult](sc, run.ArtifactPublishResult)
			if err != nil {
				return run.Result{}, err
			}
			if published.Status != wordPressPublished {
				return relinked(notLiveYet(sc.Page, published.Status), "no neighbor links to "+sc.Page.Path+" yet")
			}
			state, err := relinkStateOf(sc, checkpointRelinkNeighbors, run.ArtifactLinkContext, run.ArtifactPublishResult)
			if err != nil {
				return run.Result{}, err
			}
			around, err := neighborhoodOf(ctx, deps, sc)
			if err != nil {
				return run.Result{}, err
			}
			if !state.Planned {
				return around.planAll(ctx, deps, sc, &state)
			}
			return around.writeNext(ctx, deps, &state)
		},
	}
}

const wordPressPublished = "publish"

func relinked(result RelinkResult, message string) (run.Result, error) {
	blob, err := encode(result, "relink result")
	if err != nil {
		return run.Result{}, err
	}
	return run.Result{Artifacts: []run.Artifact{{Kind: run.ArtifactRelinkResult, Blob: blob}}, Message: message}, nil
}

func notLiveYet(page pagemap.Page, status string) RelinkResult {
	return RelinkResult{
		Neighbors: make([]NeighborResult, 0),
		Findings: []content.Finding{{
			Severity: content.SeverityWarn, Code: content.CodeTargetNotPublished,
			Message: "no neighbor links to " + page.Path + " while the site holds it as " + status +
				", since such a link would lead a visitor nowhere; its neighbors are relinked once it is published",
			Details: map[string]any{"pageId": page.ID, "path": page.Path, "status": status},
		}},
	}
}

type neighborhood struct {
	client   *wp.Client
	lc       content.LinkContext
	entity   graph.Entity
	page     pagemap.Page
	site     pagemap.Site
	index    pagemap.Index
	graph    graph.Graph
	types    pagetype.Set
	runID    string
	absolute bool
}

func neighborhoodOf(ctx context.Context, deps Deps, sc *run.StepContext) (neighborhood, error) {
	lc, err := linkContextOf(sc)
	if err != nil {
		return neighborhood{}, err
	}
	entity, err := entityOf(ctx, deps, sc)
	if err != nil {
		return neighborhood{}, err
	}
	types, err := deps.Types.Set(ctx, sc.Run.SiteID)
	if err != nil {
		return neighborhood{}, err
	}
	owner, err := deps.Sites.Get(ctx, sc.Run.SiteID)
	if err != nil {
		return neighborhood{}, err
	}
	client, err := clientFor(ctx, deps, sc.Run.SiteID)
	if err != nil {
		return neighborhood{}, err
	}

	pages, err := deps.Pages.ListBySite(ctx, sc.Run.SiteID)
	if err != nil {
		return neighborhood{}, err
	}
	entities, err := deps.Entities.ListBySite(ctx, sc.Run.SiteID)
	if err != nil {
		return neighborhood{}, err
	}
	edges, err := deps.Edges.ListBySite(ctx, sc.Run.SiteID)
	if err != nil {
		return neighborhood{}, err
	}
	built, err := graph.New(entities, edges)
	if err != nil {
		return neighborhood{}, err
	}
	absolute, err := deps.Pictures.Absolute(ctx, sc.Run.SiteID)
	if err != nil {
		return neighborhood{}, err
	}

	return neighborhood{
		client: client, lc: lc, entity: entity, page: sc.Page, site: pagemap.NewSite(owner.BaseURL),
		index: pagemap.NewIndex(pages), graph: built, types: types, runID: sc.Run.ID, absolute: absolute,
	}, nil
}

func (n neighborhood) plan(ctx context.Context, deps Deps) (RelinkResult, []relinkWrite, error) {
	result := RelinkResult{
		Neighbors: make([]NeighborResult, 0, len(n.lc.Targets)),
		Findings:  make([]content.Finding, 0),
	}
	writes := make([]relinkWrite, 0)

	candidates := n.candidates()
	for i := range candidates {
		neighbor := candidates[i].page
		if neighbor.WPID == nil || neighbor.ID == n.page.ID {
			continue
		}

		outcome, write, err := n.relinkOne(ctx, deps, neighbor)
		if err != nil {
			return RelinkResult{}, nil, err
		}
		if write != nil {
			write.Planned = candidates[i].planned
			writes = append(writes, *write)
			continue
		}
		if !candidates[i].planned && quiet(outcome) {
			continue
		}
		result.tally(outcome, n.page)
	}
	return result, writes, nil
}

func (r *RelinkResult) tally(outcome NeighborResult, page pagemap.Page) {
	r.Neighbors = append(r.Neighbors, outcome)

	switch {
	case outcome.Outcome == OutcomeLinked:
		r.Linked++
		if outcome.Sentence != "" {
			r.Findings = append(r.Findings, templatedNeighborFinding(outcome, page))
		}
	case outcome.Outcome == OutcomeConflict:
		r.Conflicts++
		r.Findings = append(r.Findings, conflictFinding(outcome))
	case outcome.Outcome == OutcomeSkipped:
		r.Skipped++
		r.Findings = append(r.Findings, skippedFinding(outcome))
	case outcome.missing:
		r.Missing++
		r.Findings = append(r.Findings, missingNeighborFinding(outcome, page))
	}
}

type candidate struct {
	page    pagemap.Page
	planned bool
}

func (n neighborhood) candidates() []candidate {
	seen := make(map[string]struct{}, len(n.lc.Targets))
	out := make([]candidate, 0, len(n.lc.Targets))
	for i := range n.lc.Targets {
		if n.lc.Targets[i].EntityID == n.entity.ID {
			continue
		}
		page, ok := n.index.ByID(n.lc.Targets[i].PageID)
		if !ok {
			continue
		}
		if _, dup := seen[page.ID]; dup {
			continue
		}
		seen[page.ID] = struct{}{}
		out = append(out, candidate{page: page, planned: true})
	}
	others := content.MayLinkTo(n.graph, n.index, n.entity.ID)
	for i := range others {
		if others[i].CanonicalPageID == nil {
			continue
		}
		page, ok := n.index.ByID(*others[i].CanonicalPageID)
		if !ok {
			continue
		}
		if _, dup := seen[page.ID]; dup {
			continue
		}
		seen[page.ID] = struct{}{}
		out = append(out, candidate{page: page})
	}
	return out
}

type owedLink struct {
	lc     content.LinkContext
	policy field.LinkPolicy
	target content.LinkTarget
}

func (n neighborhood) relinkOne(ctx context.Context, deps Deps, neighbor pagemap.Page) (NeighborResult, *relinkWrite, error) {
	outcome := NeighborResult{
		PageID: neighbor.ID, Path: neighbor.Path, Type: string(neighbor.WPType), WPID: *neighbor.WPID,
	}

	owed, reason := n.owedBy(neighbor)
	if reason != "" {
		return skip(outcome, reason), nil, nil
	}
	return n.placeOwed(ctx, deps, neighbor, owed, outcome)
}

func (n neighborhood) owedBy(neighbor pagemap.Page) (owed owedLink, standDown string) {
	if neighbor.EntityID == nil {
		return owedLink{}, ReasonNeighborUnmapped
	}
	if neighbor.WPType == pagemap.WPProductCategory {
		return owedLink{}, ReasonNeighborIsATerm
	}
	lc, policy := n.planOf(neighbor)
	target, found := lc.ByPageID(n.page.ID)
	if !found {
		return owedLink{}, owedReason(lc, n.page)
	}
	return owedLink{lc: lc, policy: policy, target: target}, ""
}

func (n neighborhood) planOf(neighbor pagemap.Page) (content.LinkContext, field.LinkPolicy) {
	policy := n.types.Policy(neighbor)
	plan := content.PlanLinks(n.graph, n.index, content.Subject{
		Site: n.site, PageID: neighbor.ID, PagePath: neighbor.Path, EntityID: *neighbor.EntityID, Absolute: n.absolute,
	}, policy)
	return plan.Context, policy
}

func (n neighborhood) placeOwed(ctx context.Context, deps Deps, neighbor pagemap.Page, owed owedLink,
	outcome NeighborResult) (NeighborResult, *relinkWrite, error) {
	body, reason, err := readSiteBody(ctx, n.client, neighbor, ReasonNeighborGone, ReasonNeighborUnreadable)
	if err != nil {
		return NeighborResult{}, nil, err
	}
	if reason != "" {
		return skip(outcome, reason), nil, nil
	}

	placement, sentence := backfill(body.doc, owed.lc, owed.policy, owed.target)
	anchor, inserted := insertedAnchor(placement)
	if !inserted {
		outcome.Outcome = OutcomeUnchanged
		outcome.Detail = firstDetail(placement)
		outcome.missing = !alreadyLinked(placement)
		return outcome, nil, nil
	}

	updated, err := body.spliced(ctx, deps, neighbor)
	if err != nil {
		return skip(outcome, ReasonNeighborUnreadable), nil, nil
	}
	outcome.Outcome = OutcomeLinked
	outcome.Anchor = anchor
	outcome.Sentence = sentence
	outcome.Before = NeighborBefore{Hash: body.raw.ContentHash, HTML: body.raw.Content}
	return outcome, &relinkWrite{Outcome: outcome, After: updated.raw, Ours: updated.ours}, nil
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

func quiet(outcome NeighborResult) bool {
	switch outcome.Outcome {
	case OutcomeSkipped:
		return outcome.Detail == ReasonNeighborOwesNothing || outcome.Detail == ReasonNeighborUnmapped ||
			outcome.Detail == ReasonNeighborOwesTheCanonicalPage
	case OutcomeUnchanged:
		return !outcome.missing
	default:
		return false
	}
}

func skip(outcome NeighborResult, reason string) NeighborResult {
	outcome.Outcome = OutcomeSkipped
	outcome.Detail = reason
	return outcome
}

func templatedNeighborFinding(outcome NeighborResult, page pagemap.Page) content.Finding {
	return content.Finding{
		Severity: content.SeverityWarn,
		Code:     CodeRelinkPhraseTemplated,
		Message: "the neighbor " + outcome.Path + " owed " + page.Path + " a link and carried no anchor for it, so " +
			"a plain sentence was added to it; rewrite it on the site if it reads poorly",
		Details: map[string]any{
			"pageId": outcome.PageID, "wpId": outcome.WPID, "sentence": outcome.Sentence, "targetPageId": page.ID,
		},
	}
}

func missingNeighborFinding(outcome NeighborResult, page pagemap.Page) content.Finding {
	return content.Finding{
		Severity: content.SeverityWarn,
		Code:     CodeNeighborLinkMissing,
		Message:  "the neighbor " + outcome.Path + " owes " + page.Path + " a link that could not be placed: " + outcome.Detail,
		Details: map[string]any{
			"pageId": outcome.PageID, "wpId": outcome.WPID, "reason": outcome.Detail, "targetPageId": page.ID,
		},
	}
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

func (n neighborhood) planAll(ctx context.Context, deps Deps, sc *run.StepContext, state *relinkState) (run.Result, error) {
	result, writes, err := n.plan(ctx, deps)
	if err != nil {
		return run.Result{}, err
	}
	state.Result, state.Writes, state.Planned = result, writes, true
	held, err := deps.reviewRelinks(ctx, sc, checkpointRelinkNeighborsReviewed, "the links "+n.page.Path+"'s neighbors owe it",
		state.bodies(n.index))
	if err != nil || held.held != nil {
		return orHeld(held, err)
	}
	waiting, err := n.settled(state, "planned the links "+strconv.Itoa(len(writes))+" neighbors of "+n.page.Path+" owe it")
	if err != nil {
		return run.Result{}, err
	}
	return withChecks(waiting, held.check), nil
}

func (n neighborhood) writeNext(ctx context.Context, deps Deps, state *relinkState) (run.Result, error) {
	write := state.Writes[state.Done]
	outcome := write.Outcome
	neighbor, known := n.index.ByID(outcome.PageID)
	if !known || neighbor.WPID == nil {
		state.Result.tally(skip(outcome, ReasonNeighborGone), n.page)
		state.Done++
		return n.settled(state, "the neighbor "+outcome.Path+" left the map before it was relinked")
	}
	landed, err := landRelinkedBody(ctx, n.client, neighbor, write.After, outcome.Before.Hash)
	if err != nil {
		check, checkErr := state.checkpoint(checkpointRelinkNeighbors)
		if checkErr != nil {
			return run.Result{}, checkErr
		}
		return run.Result{Checkpoint: check}, err
	}
	if !landed.written {
		outcome.Outcome, outcome.Detail = OutcomeConflict, "the neighbor changed on the site after its links were planned"
	} else if recorded := n.recordLanded(ctx, deps, neighbor, write, landed.hash); recorded != nil {
		return run.Result{}, recorded
	}
	state.Result.tally(outcome, n.page)
	state.Done++
	return n.settled(state, "relinked "+outcome.Path)
}

func (n neighborhood) recordLanded(ctx context.Context, deps Deps, neighbor pagemap.Page, write relinkWrite, hash string) error {
	if err := recordSpliced(ctx, deps, neighbor.ID, n.runID, splicedBody{raw: write.After, ours: write.Ours}); err != nil {
		return err
	}
	placed := []block.LinkKey{{To: n.page.Path, Anchor: write.Outcome.Anchor}}
	return adopt(ctx, deps, neighbor, n.index, n.site, write.After, hash, placed)
}

func (n neighborhood) settled(state *relinkState, message string) (run.Result, error) {
	finished := state.Done >= len(state.Writes)
	state.Finished = finished
	check, err := state.checkpoint(checkpointRelinkNeighbors)
	if err != nil {
		return run.Result{}, err
	}
	if !finished && state.Done == 0 {
		return run.Result{Next: run.TransitionWait, Checkpoint: check, Message: message}, nil
	}
	carried, err := relinked(state.Result, message)
	if err != nil {
		return run.Result{}, err
	}
	carried.Checkpoint = check
	if !finished {
		carried.Next = run.TransitionWait
	} else {
		carried.Message = "relinked " + strconv.Itoa(state.Result.Linked) + " neighbors of " + n.page.Path
	}
	return carried, nil
}
