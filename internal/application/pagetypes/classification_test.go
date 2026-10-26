package pagetypes_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type shape struct {
	level     int
	listing   bool
	ancestors []string
	wpType    pagemap.WPType
}

type shapedFacts struct {
	pages  *sqlite.PageRepo
	shapes map[string]shape
}

func (f shapedFacts) ClassificationFacts(ctx context.Context, siteID string) ([]pagetype.Facts, error) {
	stored, err := f.pages.ListBySite(ctx, siteID)
	if err != nil {
		return nil, err
	}
	byPath := make(map[string]string, len(stored))
	for i := range stored {
		byPath[stored[i].Path] = stored[i].ID
	}
	out := make([]pagetype.Facts, 0, len(stored))
	for i := range stored {
		held, shaped := f.shapes[stored[i].Path]
		if !shaped {
			continue
		}
		facts := pagetype.FactsOf(stored[i], &pagetype.Rendered{Listing: held.listing, Words: 120})
		facts.Level, facts.FrontPage, facts.Products = held.level, stored[i].Path == pagemap.RootPath, 3
		if held.wpType != "" {
			facts.WPType = held.wpType
		}
		for _, path := range held.ancestors {
			facts.Ancestors = append(facts.Ancestors, byPath[path])
		}
		out = append(out, facts)
	}
	return out, nil
}

type classifying struct {
	harness
	classification *pagetypes.Classification
	pages          *sqlite.PageRepo
	findings       *sqlite.FindingRepo
	ids            map[string]string
}

func newClassifying(t *testing.T) classifying {
	t.Helper()
	h := newHarness(t)
	pages := sqlite.NewPageRepo(h.store)
	shapes := map[string]shape{
		"/":                  {level: 0},
		"/peptides/":         {level: 1, listing: true},
		"/healing-peptides/": {level: 2, listing: true, ancestors: []string{"/peptides/"}},
		"/lab-guides/":       {level: 1},
		"/reconstitution/":   {level: 2, ancestors: []string{"/lab-guides/"}},
		"/sample-page/":      {level: 1},
		"/research/":         {level: 1},
		"/about-us/":         {level: 1},
		"/product/bpc/":      {level: 2, wpType: pagemap.WPProduct},
	}
	c := classifying{
		harness: h, pages: pages, findings: sqlite.NewFindingRepo(h.store), ids: map[string]string{},
	}
	for path := range shapes {
		c.ids[path] = sqlitetest.Page(t, h.store, h.siteID, path).ID
	}
	c.classification = pagetypes.NewClassification(pagetypes.ClassificationDeps{
		Facts: shapedFacts{pages: pages, shapes: shapes}, Types: h.service, Pages: pages, Findings: c.findings,
		UnitOfWork: h.store, Publisher: h.recorder, Clock: clock.NewFake(time.Date(2026, time.October, 26, 9, 0, 0, 0, time.UTC)),
	})
	return c
}

func (c classifying) legacy(t *testing.T, path string, preset pagetype.Preset, reason pagemap.TypeReason) {
	t.Helper()
	typeID := c.typeOf(t, c.siteID, preset).ID
	if _, err := c.pages.Classify(t.Context(), c.ids[path], pagetype.Assignment{},
		pagetype.Assignment{TypeID: typeID, Source: pagemap.TypeSourceAuto, Reason: reason}, time.Now()); err != nil {
		t.Fatalf("type %s as before 2.6: %v", path, err)
	}
}

func (c classifying) preview(t *testing.T) pagetypes.PreviewResponse {
	t.Helper()
	got, err := c.classification.Preview(t.Context(), pagetypes.PreviewRequest{SiteID: c.siteID})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	return got
}

func (c classifying) assignments(t *testing.T) map[string]pagetype.Assignment {
	t.Helper()
	out := make(map[string]pagetype.Assignment, len(c.ids))
	for path, pageID := range c.ids {
		page, err := c.pages.Get(t.Context(), pageID)
		if err != nil {
			t.Fatalf("Get %s: %v", path, err)
		}
		out[path] = pagetype.AssignmentOf(page)
	}
	return out
}

func TestThePreviewIsWhatTheApplyDoes(t *testing.T) {
	t.Parallel()
	c := newClassifying(t)
	c.legacy(t, "/peptides/", pagetype.Category, pagemap.TypeByListing)
	c.legacy(t, "/lab-guides/", pagetype.Info, pagemap.TypeByDefault)
	if err := c.pages.SetType(t.Context(), c.ids["/about-us/"], c.typeOf(t, c.siteID, pagetype.Catalog).ID, pagemap.TypeByUser, time.Now()); err != nil {
		t.Fatalf("SetType: %v", err)
	}

	preview := c.preview(t)
	if preview.Kept != 1 || preview.Moving != 8 || preview.ByDefault != 1 || preview.Hash == "" {
		t.Fatalf("the preview = %+v, want eight pages moving, the person's kept and one by default", preview)
	}
	landing, guide := c.typeOf(t, c.siteID, pagetype.Landing).ID, c.typeOf(t, c.siteID, pagetype.Guide).ID
	var fromCategory *pagetypes.PreviewGroup
	for i := range preview.Groups {
		if preview.Groups[i].From == c.typeOf(t, c.siteID, pagetype.Category).ID && preview.Groups[i].To == landing {
			fromCategory = &preview.Groups[i]
		}
	}
	if fromCategory == nil || fromCategory.Pages != 1 || !fromCategory.Retyped {
		t.Fatalf("groups = %+v, want the listing hub moving from Category to Landing", preview.Groups)
	}

	listed, err := c.classification.PreviewPages(t.Context(), pagetypes.PreviewPagesRequest{
		SiteID: c.siteID, From: "", To: guide, ListRequest: dto.ListRequest{Limit: 1},
	})
	if err != nil || len(listed.Items) != 1 || !listed.HasMore || listed.Items[0].Path != "/reconstitution/" {
		t.Fatalf("the untyped pages moving to Guide, a page at a time = %+v %v", listed, err)
	}
	next, err := c.classification.PreviewPages(t.Context(), pagetypes.PreviewPagesRequest{
		SiteID: c.siteID, From: "", To: guide, ListRequest: dto.ListRequest{Limit: 1, Cursor: string(listed.Next)},
	})
	if err != nil || len(next.Items) != 1 || next.Items[0].Path != "/research/" || next.HasMore {
		t.Fatalf("the next page = %+v %v, want /research/ and nothing after it", next, err)
	}
	kept, err := c.classification.PreviewPages(t.Context(), pagetypes.PreviewPagesRequest{SiteID: c.siteID, Status: "kept"})
	if err != nil || len(kept.Items) != 1 || kept.Items[0].Path != "/about-us/" {
		t.Fatalf("the kept pages = %+v %v", kept.Items, err)
	}

	applied, err := c.classification.Apply(t.Context(), pagetypes.ApplyRequest{SiteID: c.siteID, Hash: preview.Hash})
	if err != nil || applied.Moved != 8 || applied.Changed != 0 {
		t.Fatalf("Apply = %+v %v, want the eight pages the preview showed", applied, err)
	}
	after := c.assignments(t)
	if peptides := after["/peptides/"]; peptides.TypeID != landing || peptides.Reason != pagemap.TypeByRule || peptides.RuleID == "" {
		t.Errorf("/peptides/ after the apply = %+v, want Landing by its rule", peptides)
	}
	if about := after["/about-us/"]; about.Source != pagemap.TypeSourceManual || about.Reason != pagemap.TypeByUser {
		t.Errorf("the person's page after the apply = %+v, want it kept", about)
	}
	if again := c.preview(t); again.Moving != 0 || again.Refreshed != 0 || len(again.Groups) != 0 {
		t.Errorf("the preview after the apply = %+v, want nothing left to change", again)
	}
	open, err := c.findings.OpenBySite(t.Context(), c.siteID, finding.TypeByDefault)
	if err != nil || len(open) != 1 || *open[0].PageID != c.ids["/sample-page/"] {
		t.Errorf("type_by_default after the apply = %+v %v, want the one page no rule matched", open, err)
	}
	if len(c.recorder.Events()) == 0 {
		t.Error("no event told the window the pages moved")
	}
}

func TestAnApplyRefusesAPreviewThatMovedAndAGroupWithoutTheOneItRestsOn(t *testing.T) {
	t.Parallel()
	c := newClassifying(t)
	c.legacy(t, "/lab-guides/", pagetype.Info, pagemap.TypeByDefault)
	c.legacy(t, "/reconstitution/", pagetype.Info, pagemap.TypeByDefault)

	preview := c.preview(t)
	info, guide := c.typeOf(t, c.siteID, pagetype.Info).ID, c.typeOf(t, c.siteID, pagetype.Guide).ID
	group := pagetypes.GroupKey{From: info, To: guide}
	held := slices.IndexFunc(preview.Groups, func(g pagetypes.PreviewGroup) bool { return g.From == info && g.To == guide })
	if held < 0 || preview.Groups[held].Pages != 2 {
		t.Fatalf("groups = %+v, want both guides moving from Info to Guide", preview.Groups)
	}

	if err := c.pages.SetType(t.Context(), c.ids["/sample-page/"], info, pagemap.TypeByUser, time.Now()); err != nil {
		t.Fatalf("SetType: %v", err)
	}
	if _, err := c.classification.Apply(t.Context(), pagetypes.ApplyRequest{SiteID: c.siteID, Hash: preview.Hash, Groups: []pagetypes.GroupKey{group}}); !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("an apply over a preview that moved = %v, want CONFLICT", err)
	}

	untyped := c.preview(t)
	var under *pagetypes.PreviewGroup
	for i := range untyped.Groups {
		if untyped.Groups[i].From == "" && untyped.Groups[i].To == guide {
			under = &untyped.Groups[i]
		}
	}
	c.legacy(t, "/lab-guides/", pagetype.Info, pagemap.TypeByDefault)
	if under != nil && len(under.DependsOn) > 0 {
		t.Fatalf("an untyped group = %+v, want no dependency here", under)
	}
	if _, err := c.pages.Classify(t.Context(), c.ids["/reconstitution/"], pagetype.Assignment{TypeID: info, Source: pagemap.TypeSourceAuto, Reason: pagemap.TypeByDefault},
		pagetype.Assignment{TypeID: c.typeOf(t, c.siteID, pagetype.Landing).ID, Source: pagemap.TypeSourceAuto, Reason: pagemap.TypeByDefault}, time.Now()); err != nil {
		t.Fatalf("move /reconstitution/ to another group: %v", err)
	}
	split := c.preview(t)
	child := pagetypes.GroupKey{From: c.typeOf(t, c.siteID, pagetype.Landing).ID, To: guide}
	at := slices.IndexFunc(split.Groups, func(g pagetypes.PreviewGroup) bool { return g.From == child.From && g.To == child.To })
	if at < 0 || !slices.Contains(split.Groups[at].DependsOn, group) {
		t.Fatalf("groups = %+v, want the page under the guides to rest on the guides' group", split.Groups)
	}
	_, err := c.classification.Apply(t.Context(), pagetypes.ApplyRequest{SiteID: c.siteID, Hash: split.Hash, Groups: []pagetypes.GroupKey{child}})
	if name, reason := reasonOf(t, err); name != "groups" || reason != "group_depends" {
		t.Fatalf("applying the child's group alone = %v (%s %s), want it refused", err, name, reason)
	}
	if _, err = c.classification.Apply(t.Context(), pagetypes.ApplyRequest{SiteID: c.siteID, Hash: split.Hash, Groups: []pagetypes.GroupKey{child, group}}); err != nil {
		t.Fatalf("applying both: %v", err)
	}
	if got := c.assignments(t)["/reconstitution/"]; got.TypeID != guide {
		t.Errorf("/reconstitution/ = %+v, want Guide", got)
	}
}

func TestARuleIsTestedAgainstOnePageAndSaysWhy(t *testing.T) {
	t.Parallel()
	c := newClassifying(t)
	rules := c.rules(t, c.siteID)

	got, err := c.classification.TestRule(t.Context(), pagetypes.TestRuleRequest{SiteID: c.siteID, PageID: c.ids["/lab-guides/"], RuleID: rules.Rules[5].ID})
	if err != nil {
		t.Fatalf("TestRule: %v", err)
	}
	if got.Verdict != string(pagetype.VerdictUnmet) || got.Facts.Level != 1 || got.Facts.Listing || got.Outcome.TypeID != c.typeOf(t, c.siteID, pagetype.Guide).ID {
		t.Fatalf("rule 6 on /lab-guides/ = %+v, want unmet with the page's facts and its own outcome", got)
	}
	if !slices.Contains(got.Conditions, pagetype.ConditionResult{Condition: pagetype.ConditionListing, Verdict: pagetype.VerdictUnmet}) {
		t.Errorf("the conditions = %+v, want the listing named as what failed", got.Conditions)
	}

	draft := &pagetypes.SaveRuleRequest{Name: "Labs", TypeID: c.typeOf(t, c.siteID, pagetype.Info).ID, Conditions: pagetype.Conditions{
		URLs: []pagetype.URLMatch{{Mode: pagetype.URLGlob, Value: "/lab-*/"}},
	}}
	if got, err = c.classification.TestRule(t.Context(), pagetypes.TestRuleRequest{SiteID: c.siteID, PageID: c.ids["/lab-guides/"], Rule: draft}); err != nil || got.Verdict != string(pagetype.VerdictMet) {
		t.Fatalf("a draft rule = %+v %v, want it met", got, err)
	}
	draft.Conditions.URLs[0].Value = "/lab-[x]/"
	if _, err = c.classification.TestRule(t.Context(), pagetypes.TestRuleRequest{SiteID: c.siteID, PageID: c.ids["/lab-guides/"], Rule: draft}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a draft with a bad pattern = %v, want INVALID", err)
	}

	matching, err := c.classification.Matching(t.Context(), c.siteID, rules.Rules[12].ID)
	if err != nil || !slices.Equal(matching, []string{c.ids["/about-us/"]}) {
		t.Errorf("the pages the slug rule matches = %v %v", matching, err)
	}
}
