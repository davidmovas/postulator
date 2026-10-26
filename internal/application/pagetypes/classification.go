package pagetypes

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	ApplyBatch          = 1000
	reasonPreviewMoved  = "preview_moved"
	reasonGroupDepends  = "group_depends"
	previewStatusKept   = "kept"
	previewStatusWaited = "waiting"
	draftRuleID         = "draft"
)

type factsReader interface {
	ClassificationFacts(ctx context.Context, siteID string) ([]pagetype.Facts, error)
}

type classifiedTypes interface {
	Set(ctx context.Context, siteID string) (pagetype.Set, error)
	Rules(ctx context.Context, siteID string) ([]pagetype.Rule, error)
}

type classifiedPages interface {
	Classify(ctx context.Context, pageID string, was, now pagetype.Assignment, at time.Time) (bool, error)
}

type findingWriter interface {
	Upsert(ctx context.Context, record finding.Finding) error
	Clear(ctx context.Context, siteID string, code finding.Code, subjectKey, resolvedBy, resolution string, at time.Time) error
}

type ClassificationDeps struct {
	Facts      factsReader
	Types      classifiedTypes
	Pages      classifiedPages
	Findings   findingWriter
	UnitOfWork unitOfWork
	Publisher  application.Publisher
	Clock      clock.Clock
}

type Classification struct {
	deps ClassificationDeps
}

func NewClassification(deps ClassificationDeps) *Classification {
	return &Classification{deps: deps}
}

type classified struct {
	set      pagetype.Set
	facts    []pagetype.Facts
	outcomes []pagetype.Outcome
	rules    []pagetype.Rule
}

func (c *Classification) classify(ctx context.Context, siteID string) (classified, error) {
	if strings.TrimSpace(siteID) == "" {
		return classified{}, errors.New(errors.Invalid, "a site is required").WithDetail("field", "siteId")
	}
	set, err := c.deps.Types.Set(ctx, siteID)
	if err != nil {
		return classified{}, err
	}
	rules, err := c.deps.Types.Rules(ctx, siteID)
	if err != nil {
		return classified{}, err
	}
	facts, err := c.deps.Facts.ClassificationFacts(ctx, siteID)
	if err != nil {
		return classified{}, err
	}
	outcomes := pagetype.NewClassifier(rules, set).Classify(facts, pagetype.ModePreview)
	return classified{set: set, facts: facts, outcomes: outcomes, rules: rules}, nil
}

func keyOf(outcome *pagetype.Outcome) GroupKey {
	return GroupKey{From: outcome.Current.TypeID, To: outcome.TypeID}
}

func (s classified) groups() []PreviewGroup {
	byPage := make(map[string]*pagetype.Outcome, len(s.outcomes))
	for i := range s.outcomes {
		byPage[s.outcomes[i].PageID] = &s.outcomes[i]
	}
	held := make(map[GroupKey]*PreviewGroup)
	rules := make(map[GroupKey]map[string]int)
	needs := make(map[GroupKey]map[GroupKey]bool)
	for i := range s.outcomes {
		outcome := &s.outcomes[i]
		if !outcome.Changed() {
			continue
		}
		key := keyOf(outcome)
		group, known := held[key]
		if !known {
			group = &PreviewGroup{From: key.From, To: key.To, Retyped: key.From != key.To, Rules: []RuleCount{}, DependsOn: []GroupKey{}}
			held[key], rules[key], needs[key] = group, map[string]int{}, map[GroupKey]bool{}
		}
		group.Pages++
		rules[key][outcome.RuleID]++
		for _, pageID := range outcome.Depends {
			if above, found := byPage[pageID]; found && above.Changed() && keyOf(above) != key {
				needs[key][keyOf(above)] = true
			}
		}
	}
	out := make([]PreviewGroup, 0, len(held))
	for key, group := range held {
		for ruleID, pages := range rules[key] {
			group.Rules = append(group.Rules, RuleCount{RuleID: ruleID, Pages: pages})
		}
		slices.SortFunc(group.Rules, func(a, b RuleCount) int {
			return cmp.Or(cmp.Compare(b.Pages, a.Pages), strings.Compare(a.RuleID, b.RuleID))
		})
		for need := range needs[key] {
			group.DependsOn = append(group.DependsOn, need)
		}
		slices.SortFunc(group.DependsOn, compareKeys)
		out = append(out, *group)
	}
	slices.SortFunc(out, func(a, b PreviewGroup) int {
		return cmp.Or(cmp.Compare(b.Pages, a.Pages), compareKeys(GroupKey{From: a.From, To: a.To}, GroupKey{From: b.From, To: b.To}))
	})
	return out
}

func compareKeys(a, b GroupKey) int {
	return cmp.Or(strings.Compare(a.From, b.From), strings.Compare(a.To, b.To))
}

func (s classified) hash() string {
	lines := make([]string, 0, len(s.outcomes))
	for i := range s.outcomes {
		outcome := &s.outcomes[i]
		if outcome.Changed() {
			lines = append(lines, strings.Join([]string{
				outcome.PageID, outcome.Current.TypeID, string(outcome.Current.Reason), outcome.Current.RuleID,
				outcome.TypeID, string(outcome.Reason), outcome.RuleID,
			}, "|"))
		}
	}
	slices.Sort(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:])
}

func (c *Classification) Preview(ctx context.Context, req PreviewRequest) (PreviewResponse, error) {
	state, err := c.classify(ctx, req.SiteID)
	if err != nil {
		return PreviewResponse{}, err
	}
	out := PreviewResponse{SiteID: req.SiteID, Hash: state.hash(), Groups: state.groups()}
	for i := range state.outcomes {
		outcome := &state.outcomes[i]
		switch {
		case !outcome.Eligible:
			out.Kept++
		case outcome.Waiting:
			out.Waiting++
		case outcome.Retyped():
			out.Moving++
		case outcome.Changed():
			out.Refreshed++
		default:
			out.Unchanged++
		}
		if outcome.Eligible && !outcome.Waiting && outcome.Reason == pagemap.TypeByUnmatched {
			out.ByDefault++
		}
	}
	return out, nil
}

func (c *Classification) PreviewPages(ctx context.Context, req PreviewPagesRequest) (paging.List[PreviewPage], error) {
	state, err := c.classify(ctx, req.SiteID)
	if err != nil {
		return paging.List[PreviewPage]{}, err
	}
	paths := make(map[string]string, len(state.facts))
	for i := range state.facts {
		paths[state.facts[i].PageID] = state.facts[i].Path
	}
	rows := make([]PreviewPage, 0)
	for i := range state.outcomes {
		outcome := &state.outcomes[i]
		var shown bool
		switch req.Status {
		case previewStatusKept:
			shown = !outcome.Eligible
		case previewStatusWaited:
			shown = outcome.Eligible && outcome.Waiting
		default:
			shown = outcome.Changed() && keyOf(outcome) == GroupKey{From: req.From, To: req.To}
		}
		if shown {
			rows = append(rows, PreviewPage{
				PageID: outcome.PageID, Path: paths[outcome.PageID], From: outcome.Current.TypeID, To: outcome.TypeID,
				RuleID: outcome.RuleID, Reason: string(outcome.Reason), Was: string(outcome.Current.Reason), WasRuleID: outcome.Current.RuleID,
			})
		}
	}
	slices.SortFunc(rows, func(a, b PreviewPage) int {
		return cmp.Or(strings.Compare(a.Path, b.Path), strings.Compare(a.PageID, b.PageID))
	})
	keyset := paging.Keyset[PreviewPage]{
		IDColumn: "page_id", ID: func(row PreviewPage) string { return row.PageID },
		Keys: []paging.SortKey[PreviewPage]{paging.TextKey("path", "path", func(row PreviewPage) any { return row.Path })},
	}
	return keyset.Page(rows, paging.Request{After: paging.Cursor(req.Cursor), Limit: dto.PageSize(req.Limit)})
}

func (c *Classification) Apply(ctx context.Context, req ApplyRequest) (ApplyResponse, error) {
	state, err := c.classify(ctx, req.SiteID)
	if err != nil {
		return ApplyResponse{}, err
	}
	if req.Hash != state.hash() {
		return ApplyResponse{}, errors.New(errors.Conflict, "the rules or the site changed since the preview; look at it again before applying").
			WithDetail("field", "hash").WithDetail("reason", reasonPreviewMoved)
	}
	chosen, err := chosenGroups(state.groups(), req.Groups)
	if err != nil {
		return ApplyResponse{}, err
	}
	pending := make([]*pagetype.Outcome, 0)
	for i := range state.outcomes {
		if outcome := &state.outcomes[i]; outcome.Changed() && chosen[keyOf(outcome)] {
			pending = append(pending, outcome)
		}
	}
	names := make(map[string]string, len(state.facts))
	for i := range state.facts {
		names[state.facts[i].PageID] = state.facts[i].Path
	}
	var out ApplyResponse
	for start := 0; start < len(pending); start += ApplyBatch {
		batch := pending[start:min(start+ApplyBatch, len(pending))]
		err = c.deps.UnitOfWork.Do(ctx, func(u context.Context) error {
			moved, changed, batchErr := c.applyBatch(u, req.SiteID, batch, state.set, names)
			out.Moved += moved
			out.Changed += changed
			return batchErr
		})
		if err != nil {
			return out, err
		}
	}
	if c.deps.Publisher != nil && out.Moved > 0 {
		if publishErr := c.deps.Publisher.Publish(events.PagesChanged, events.PagesChangedPayload{SiteID: req.SiteID}); publishErr != nil {
			return out, publishErr
		}
	}
	return out, nil
}

func chosenGroups(groups []PreviewGroup, asked []GroupKey) (map[GroupKey]bool, error) {
	chosen := make(map[GroupKey]bool, len(groups))
	if len(asked) == 0 {
		for i := range groups {
			chosen[GroupKey{From: groups[i].From, To: groups[i].To}] = true
		}
		return chosen, nil
	}
	known := make(map[GroupKey]*PreviewGroup, len(groups))
	for i := range groups {
		known[GroupKey{From: groups[i].From, To: groups[i].To}] = &groups[i]
	}
	for _, key := range asked {
		if _, held := known[key]; !held {
			return nil, errors.New(errors.Invalid, "a group to apply is not in the preview; look at it again").
				WithDetail("field", "groups").WithDetail("reason", reasonPreviewMoved)
		}
		chosen[key] = true
	}
	for key := range chosen {
		for _, need := range known[key].DependsOn {
			if !chosen[need] {
				return nil, errors.New(errors.Invalid, "a chosen group rests on pages another group moves; choose that group too").
					WithDetail("field", "groups").WithDetail("reason", reasonGroupDepends).WithDetail("from", need.From).WithDetail("to", need.To)
			}
		}
	}
	return chosen, nil
}

func (c *Classification) applyBatch(ctx context.Context, siteID string, batch []*pagetype.Outcome, set pagetype.Set,
	paths map[string]string) (moved, changed int, err error) {
	now := c.deps.Clock.Now().UTC().Truncate(time.Second)
	for _, outcome := range batch {
		placed := pagetype.Assignment{TypeID: outcome.TypeID, Source: pagemap.TypeSourceAuto, Reason: outcome.Reason, RuleID: outcome.RuleID}
		written, writeErr := c.deps.Pages.Classify(ctx, outcome.PageID, outcome.Current, placed, now)
		if writeErr != nil {
			return moved, changed, writeErr
		}
		if !written {
			changed++
			continue
		}
		moved++
		if flagErr := c.flag(ctx, siteID, outcome, set, paths, now); flagErr != nil {
			return moved, changed, flagErr
		}
	}
	return moved, changed, nil
}

func (c *Classification) flag(ctx context.Context, siteID string, outcome *pagetype.Outcome, set pagetype.Set,
	paths map[string]string, now time.Time) error {
	subject := finding.SubjectPage(outcome.PageID)
	if outcome.Reason != pagemap.TypeByUnmatched {
		return c.deps.Findings.Clear(ctx, siteID, finding.TypeByDefault, subject, finding.ResolvedByModel, finding.ResolutionPageTyped, now)
	}
	kind, _ := set.ByID(outcome.TypeID)
	encoded, err := json.Marshal(map[string]string{"path": paths[outcome.PageID], "type": kind.Name})
	if err != nil {
		return err
	}
	pageID := outcome.PageID
	record, err := finding.New(finding.Finding{
		ID: id.New(), SiteID: siteID, PageID: &pageID, Code: finding.TypeByDefault, SubjectKey: subject, Subject: encoded,
		CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return err
	}
	return c.deps.Findings.Upsert(ctx, record)
}

func (c *Classification) TestRule(ctx context.Context, req TestRuleRequest) (TestRuleResponse, error) {
	state, err := c.classify(ctx, req.SiteID)
	if err != nil {
		return TestRuleResponse{}, err
	}
	rule, err := testedRule(req, state)
	if err != nil {
		return TestRuleResponse{}, err
	}
	at := slices.IndexFunc(state.facts, func(f pagetype.Facts) bool { return f.PageID == req.PageID })
	if at < 0 {
		return TestRuleResponse{}, errors.New(errors.NotFound, "the page is not one the rules classify: it is planned, archived or of another site").
			WithDetail("field", "pageId")
	}
	explained := pagetype.NewClassifier(state.rules, state.set).Explain(rule, req.PageID, state.facts, pagetype.ModePreview)
	return TestRuleResponse{
		Verdict: string(explained.Verdict), Conditions: explained.Conditions, Facts: factsView(&state.facts[at]),
		Outcome: OutcomeView{
			TypeID: explained.Outcome.TypeID, RuleID: explained.Outcome.RuleID, Reason: string(explained.Outcome.Reason),
			Waiting: explained.Outcome.Waiting, Kept: !explained.Outcome.Eligible,
		},
	}, nil
}

func testedRule(req TestRuleRequest, state classified) (pagetype.Rule, error) {
	if req.Rule == nil {
		at := slices.IndexFunc(state.rules, func(rule pagetype.Rule) bool { return rule.ID == req.RuleID })
		if at < 0 {
			return pagetype.Rule{}, errors.New(errors.NotFound, "classification rule not found").WithDetail("ruleId", req.RuleID)
		}
		return state.rules[at], nil
	}
	rule := pagetype.Rule{
		ID: draftRuleID, SiteID: req.SiteID, Kind: pagetype.KindRule, Name: strings.TrimSpace(req.Rule.Name), Enabled: true,
		TypeID: strings.TrimSpace(req.Rule.TypeID), Conditions: req.Rule.Conditions.Normalized(),
	}
	if rule.Name == "" {
		rule.Name = draftRuleID
	}
	return rule, rule.Validate(state.set)
}

func (c *Classification) Matching(ctx context.Context, siteID, ruleID string) ([]string, error) {
	state, err := c.classify(ctx, siteID)
	if err != nil {
		return nil, err
	}
	at := slices.IndexFunc(state.rules, func(rule pagetype.Rule) bool { return rule.ID == ruleID })
	if at < 0 {
		return nil, errors.New(errors.NotFound, "classification rule not found").WithDetail("ruleId", ruleID)
	}
	return pagetype.NewClassifier(state.rules, state.set).Matches(state.rules[at], state.facts, pagetype.ModePreview), nil
}

func factsView(f *pagetype.Facts) FactsView {
	return FactsView{
		PageID: f.PageID, Path: f.Path, Slug: f.Slug, PostType: string(f.WPType), Technical: string(f.Technical),
		FrontPage: f.FrontPage, Rendered: f.Rendered, Listing: f.Listing, Products: f.Products, Schema: slices.Clone(f.Schema),
		Words: f.Words, H2: f.H2, Level: f.Level, Ancestors: slices.Clone(f.Ancestors),
	}
}
