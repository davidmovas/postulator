package steps

import (
	"context"
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
	ReasonNeighborNoTemplate = "the neighbor has no template, so nothing says what it may link to"
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
			around, err := neighborhoodOf(ctx, deps, sc)
			if err != nil {
				return run.Result{}, err
			}
			result, err := around.relink(ctx, deps)
			if err != nil {
				return run.Result{}, err
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

type neighborhood struct {
	client *wp.Client
	lc     content.LinkContext
	entity graph.Entity
	page   pagemap.Page
	site   pagemap.Site
	index  pagemap.Index
	graph  graph.Graph
	base   template.LinkPolicy
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
	base, err := effectivePolicy(ctx, deps, sc.Run.SiteID, template.TemplateSpec{})
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

	return neighborhood{
		client: client, lc: lc, entity: entity, page: sc.Page, site: pagemap.NewSite(owner.BaseURL),
		index: pagemap.NewIndex(pages), graph: built, base: base,
	}, nil
}

func (n neighborhood) relink(ctx context.Context, deps Deps) (RelinkResult, error) {
	result := RelinkResult{
		Neighbors: make([]NeighborResult, 0, len(n.lc.Targets)),
		Findings:  make([]content.Finding, 0),
	}

	candidates := n.candidates()
	for i := range candidates {
		neighbor := candidates[i].page
		if neighbor.WPID == nil || neighbor.ID == n.page.ID {
			continue
		}

		outcome, err := n.relinkOne(ctx, deps, neighbor)
		if err != nil {
			return RelinkResult{}, err
		}
		if !candidates[i].planned && quiet(outcome) {
			continue
		}
		result.tally(outcome, n.page)
	}
	return result, nil
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
	policy template.LinkPolicy
	target content.LinkTarget
}

func (n neighborhood) relinkOne(ctx context.Context, deps Deps, neighbor pagemap.Page) (NeighborResult, error) {
	outcome := NeighborResult{
		PageID: neighbor.ID, Path: neighbor.Path, Type: string(neighbor.WPType), WPID: *neighbor.WPID,
	}

	owed, reason, err := n.owedBy(ctx, deps, neighbor)
	if err != nil {
		return NeighborResult{}, err
	}
	if reason != "" {
		return skip(outcome, reason), nil
	}
	return n.placeOwed(ctx, deps, neighbor, owed, outcome)
}

func (n neighborhood) owedBy(ctx context.Context, deps Deps, neighbor pagemap.Page) (owed owedLink, standDown string, err error) {
	if neighbor.EntityID == nil {
		return owedLink{}, ReasonNeighborUnmapped, nil
	}
	lc, policy, err := n.planOf(ctx, deps, neighbor)
	if errors.IsCode(err, errors.NotFound) {
		return owedLink{}, ReasonNeighborNoTemplate, nil
	}
	if err != nil {
		return owedLink{}, "", err
	}

	target, found := lc.ByPageID(n.page.ID)
	if !found {
		return owedLink{}, owedReason(lc, n.page), nil
	}
	if neighbor.WPType == pagemap.WPProductCategory {
		return owedLink{}, ReasonNeighborIsATerm, nil
	}
	return owedLink{lc: lc, policy: policy, target: target}, "", nil
}

func (n neighborhood) planOf(ctx context.Context, deps Deps, neighbor pagemap.Page) (content.LinkContext, template.LinkPolicy, error) {
	resolved, err := deps.Policies.ResolveForPage(ctx, templates.ResolveForPageRequest{PageID: neighbor.ID})
	if err != nil {
		return content.LinkContext{}, template.LinkPolicy{}, err
	}

	policy := n.base
	policy.Rules = templates.EffectiveRules(n.base.Rules, resolved.Spec.LinkRules)

	plan := content.PlanLinks(n.graph, n.index, content.Subject{
		Site: n.site, PageID: neighbor.ID, PagePath: neighbor.Path, EntityID: *neighbor.EntityID,
	}, policy)
	return plan.Context, policy, nil
}

func (n neighborhood) placeOwed(ctx context.Context, deps Deps, neighbor pagemap.Page, owed owedLink,
	outcome NeighborResult) (NeighborResult, error) {
	body, reason, err := readSiteBody(ctx, n.client, neighbor, ReasonNeighborGone, ReasonNeighborUnreadable)
	if err != nil {
		return NeighborResult{}, err
	}
	if reason != "" {
		return skip(outcome, reason), nil
	}

	placement, sentence := backfill(body.doc, owed.lc, owed.policy, owed.target)
	anchor, inserted := insertedAnchor(placement)
	if !inserted {
		outcome.Outcome = OutcomeUnchanged
		outcome.Detail = firstDetail(placement)
		outcome.missing = !alreadyLinked(placement)
		return outcome, nil
	}

	updated, err := body.doc.Render()
	if err != nil {
		return skip(outcome, ReasonNeighborUnreadable), nil
	}
	hash, err := body.put(ctx, n.client, neighbor, updated)
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
	outcome.Sentence = sentence
	outcome.Before = NeighborBefore{Hash: body.raw.ContentHash, HTML: body.raw.Content}
	if adoptErr := adopt(ctx, deps, neighbor, n.index, n.site, body.doc, hash); adoptErr != nil {
		return NeighborResult{}, adoptErr
	}
	return outcome, nil
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
		return outcome.Detail == ReasonNeighborOwesNothing || outcome.Detail == ReasonNeighborNoTemplate ||
			outcome.Detail == ReasonNeighborUnmapped || outcome.Detail == ReasonNeighborOwesTheCanonicalPage
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
