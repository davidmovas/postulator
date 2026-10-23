package sqlite

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

var findingStamp = time.Date(2026, time.October, 17, 9, 0, 0, 0, time.UTC)

func findingStore(t *testing.T) *Store {
	t.Helper()

	store := openStore(t, nil)
	at := formatTime(findingStamp)
	for _, setup := range []string{
		`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`,
		`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s2', 'Other', 'https://other', 'site:s2:wp_password', 'active', ?, ?)`,
		`INSERT INTO pages (id, site_id, path, slug, wp_type, status, created_at, updated_at) VALUES ('pg1', 's1', '/tb-500/', 'tb-500', 'product', 'exists', ?, ?)`,
		`INSERT INTO pages (id, site_id, path, slug, wp_type, status, created_at, updated_at) VALUES ('pg2', 's1', '/news/', 'news', 'post', 'exists', ?, ?)`,
	} {
		if _, err := store.writer.ExecContext(t.Context(), setup, at, at); err != nil {
			t.Fatalf("%s: %v", setup, err)
		}
	}
	return store
}

func newFinding(t *testing.T, siteID string, pageID *string, code finding.Code, key, subject string, at time.Time) finding.Finding {
	t.Helper()

	f, err := finding.New(finding.Finding{
		ID: id.New(), SiteID: siteID, PageID: pageID, Code: code, SubjectKey: key, Subject: json.RawMessage(subject), CreatedAt: at, UpdatedAt: at,
	})
	if err != nil {
		t.Fatalf("finding.New: %v", err)
	}
	return f
}

func TestFindingRepoUpsertFollowsTheLifecycle(t *testing.T) {
	t.Parallel()

	store := findingStore(t)
	repo := NewFindingRepo(store)
	page := "pg1"
	key := finding.SubjectCrumb(page, "/peptides/")
	at := func(minutes int) time.Time { return findingStamp.Add(time.Duration(minutes) * time.Minute) }
	requireOne := func(when string, want finding.Finding) {
		t.Helper()
		got, err := repo.ListForPage(t.Context(), page)
		if err != nil || !reflect.DeepEqual(got, []finding.Finding{want}) {
			t.Fatalf("after %s the page's findings are %+v, %v\nwant %+v", when, got, err, want)
		}
	}

	first := newFinding(t, "s1", &page, finding.ParentUnresolved, key, `{"path":"/peptides/"}`, at(0))
	if err := repo.Upsert(t.Context(), first); err != nil {
		t.Fatalf("Upsert a new finding: %v", err)
	}
	want := first
	requireOne("a new finding", want)

	refreshed := newFinding(t, "s1", &page, finding.ParentUnresolved, key, `{"path":"/peptides/","name":"Peptides"}`, at(1))
	if err := repo.Upsert(t.Context(), refreshed); err != nil {
		t.Fatalf("Upsert an open finding: %v", err)
	}
	want.Subject, want.UpdatedAt = refreshed.Subject, refreshed.UpdatedAt
	requireOne("refreshing an open finding", want)

	if err := repo.Clear(t.Context(), "s1", finding.ParentUnresolved, key, finding.ResolvedByObservation, finding.ResolutionObservedClear, at(2)); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	want.Status, want.Resolution, want.ResolvedBy, want.UpdatedAt = finding.Resolved, finding.ResolutionObservedClear, finding.ResolvedByObservation, at(2)
	requireOne("clearing it", want)
	if err := repo.Clear(t.Context(), "s1", finding.ParentUnresolved, key, "someone", "again", at(3)); err != nil {
		t.Fatalf("Clear a resolved finding: %v", err)
	}
	if err := repo.Clear(t.Context(), "s1", finding.ParentDiffers, key, finding.ResolvedByObservation, finding.ResolutionObservedClear, at(3)); err != nil {
		t.Fatalf("Clear a finding never raised: %v", err)
	}
	requireOne("clearing what is not open", want)

	reopened := newFinding(t, "s1", &page, finding.ParentUnresolved, key, `{"path":"/peptides/","name":"Peptides"}`, at(4))
	if err := repo.Upsert(t.Context(), reopened); err != nil {
		t.Fatalf("Upsert a resolved finding: %v", err)
	}
	want.Status, want.Resolution, want.ResolvedBy, want.UpdatedAt = finding.Open, "", "", at(4)
	requireOne("raising a resolved finding again", want)

	if _, err := store.writer.ExecContext(t.Context(), `UPDATE findings SET status = 'dismissed', resolution = 'ignore', resolved_by = 'person' WHERE id = ?`, first.ID); err != nil {
		t.Fatalf("dismiss the finding: %v", err)
	}
	want.Status, want.Resolution, want.ResolvedBy = finding.Dismissed, "ignore", "person"
	same := newFinding(t, "s1", &page, finding.ParentUnresolved, key, `{ "path": "/peptides/", "name": "Peptides" }`, at(5))
	if err := repo.Upsert(t.Context(), same); err != nil {
		t.Fatalf("Upsert a dismissed finding with the same subject: %v", err)
	}
	requireOne("raising a dismissed finding with the same subject", want)

	changed := newFinding(t, "s1", &page, finding.ParentUnresolved, key, `{"path":"/peptides/","name":"Peptide shop"}`, at(6))
	if err := repo.Upsert(t.Context(), changed); err != nil {
		t.Fatalf("Upsert a dismissed finding with another subject: %v", err)
	}
	want.Status, want.Resolution, want.ResolvedBy, want.Subject, want.UpdatedAt = finding.Open, "", "", changed.Subject, at(6)
	requireOne("raising a dismissed finding whose subject changed", want)

	unknown := changed
	unknown.Code = "made_up"
	if err := repo.Upsert(t.Context(), unknown); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("an unknown code code = %q, want INVALID", errors.CodeOf(err))
	}
	stray := newFinding(t, "s9", nil, finding.PluginOutdated, finding.SubjectSite, `{}`, at(7))
	if err := repo.Upsert(t.Context(), stray); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a finding of no site code = %q, want INVALID", errors.CodeOf(err))
	}
}

func TestAFindingAPersonResolvedStaysResolvedWhileItsFactsStand(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		resolvedBy string
		subject    string
		want       finding.Status
	}{
		{name: "a person's resolution and the same facts", resolvedBy: finding.ResolvedByPerson, subject: `{"workbook":["Peptides"], "site":["Shop"]}`, want: finding.Resolved},
		{name: "the agent's resolution and the same facts", resolvedBy: finding.ResolvedByAgent, subject: `{"workbook":["Peptides"],"site":["Shop"]}`, want: finding.Resolved},
		{name: "a person's resolution and facts that changed", resolvedBy: finding.ResolvedByPerson, subject: `{"workbook":["Peptides"],"site":["Blends"]}`, want: finding.Open},
		{name: "the observation's close and the cause back", resolvedBy: finding.ResolvedByObservation, subject: `{"workbook":["Peptides"],"site":["Shop"]}`, want: finding.Open},
		{name: "an import's close and the cause back", resolvedBy: finding.ResolvedByImport, subject: `{"workbook":["Peptides"],"site":["Shop"]}`, want: finding.Open},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := findingStore(t)
			repo := NewFindingRepo(store)
			page := "pg1"
			raised := newFinding(t, "s1", &page, finding.ParentDiffers, finding.SubjectPage(page), `{"workbook":["Peptides"],"site":["Shop"]}`, findingStamp)
			if err := repo.Upsert(t.Context(), raised); err != nil {
				t.Fatalf("Upsert: %v", err)
			}
			if err := repo.Settle(t.Context(), raised.ID, finding.Resolved, "accept_site", tc.resolvedBy, findingStamp.Add(time.Minute)); err != nil {
				t.Fatalf("Settle: %v", err)
			}
			again := newFinding(t, "s1", &page, finding.ParentDiffers, finding.SubjectPage(page), tc.subject, findingStamp.Add(time.Hour))
			if err := repo.Upsert(t.Context(), again); err != nil {
				t.Fatalf("Upsert again: %v", err)
			}
			got, err := repo.Get(t.Context(), raised.ID)
			if err != nil || got.Status != tc.want {
				t.Fatalf("after it was raised again the finding is %+v, %v; want %s", got, err, tc.want)
			}
		})
	}
}

func TestAFindingMovesOntoAPageOfItsSite(t *testing.T) {
	t.Parallel()

	store := findingStore(t)
	if _, err := store.writer.ExecContext(t.Context(),
		`INSERT INTO pages (id, site_id, path, slug, wp_type, status, created_at, updated_at) VALUES ('pg3', 's2', '/x/', 'x', 'page', 'exists', ?, ?)`,
		formatTime(findingStamp), formatTime(findingStamp)); err != nil {
		t.Fatalf("insert another site's page: %v", err)
	}
	repo := NewFindingRepo(store)
	planned := "pg2"
	row := newFinding(t, "s1", &planned, finding.RowWithoutPage, "row::guide", `{"sheet":"","plannedPath":"/news/","product":false}`, findingStamp)
	if err := repo.Upsert(t.Context(), row); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	later := findingStamp.Add(time.Hour)
	if err := repo.MoveToPage(t.Context(), row.ID, "pg1", later); err != nil {
		t.Fatalf("MoveToPage: %v", err)
	}
	want := row
	moved := "pg1"
	want.PageID, want.UpdatedAt = &moved, later
	if got, err := repo.Get(t.Context(), row.ID); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("the moved finding = %+v, %v\nwant %+v", got, err, want)
	}

	cases := []struct {
		name      string
		findingID string
		pageID    string
	}{
		{name: "a page of another site", findingID: row.ID, pageID: "pg3"},
		{name: "an unknown page", findingID: row.ID, pageID: "missing"},
		{name: "an unknown finding", findingID: "missing", pageID: "pg1"},
	}
	for _, tc := range cases {
		if err := repo.MoveToPage(t.Context(), tc.findingID, tc.pageID, later); !errors.IsCode(err, errors.NotFound) {
			t.Errorf("%s: MoveToPage = %v, want NotFound", tc.name, err)
		}
	}
	if got, err := repo.Get(t.Context(), row.ID); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("the finding after the refusals = %+v, %v; want it where it was moved", got, err)
	}
}

func TestAPersonSettlesAnOpenFindingOnce(t *testing.T) {
	t.Parallel()

	store := findingStore(t)
	repo := NewFindingRepo(store)
	proposed := newFinding(t, "s1", nil, finding.EntityCardProposed, finding.SubjectEntityCard("e1"), `{"entityId":"e1"}`, findingStamp)
	if err := repo.Upsert(t.Context(), proposed); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	read, err := repo.Get(t.Context(), proposed.ID)
	if err != nil || !reflect.DeepEqual(read, proposed) {
		t.Fatalf("Get = %+v, %v; want %+v", read, err, proposed)
	}

	later := findingStamp.Add(time.Hour)
	if err = repo.Settle(t.Context(), proposed.ID, finding.Resolved, "accepted", "person", later); err != nil {
		t.Fatalf("Settle: %v", err)
	}
	settled, err := repo.Get(t.Context(), proposed.ID)
	if err != nil || settled.Status != finding.Resolved || settled.Resolution != "accepted" || settled.ResolvedBy != "person" || !settled.UpdatedAt.Equal(later) {
		t.Fatalf("after Settle = %+v, %v", settled, err)
	}
	if err = repo.Settle(t.Context(), proposed.ID, finding.Dismissed, "ignored", "person", later); !errors.IsCode(err, errors.Conflict) {
		t.Errorf("settling a settled finding = %v, want Conflict", err)
	}
	if err = repo.Settle(t.Context(), "missing", finding.Dismissed, "ignored", "person", later); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("settling an unknown finding = %v, want NotFound", err)
	}
	if err = repo.Settle(t.Context(), proposed.ID, finding.Open, "", "person", later); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("settling a finding open = %v, want Invalid", err)
	}
	if _, err = repo.Get(t.Context(), "missing"); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Get of an unknown finding = %v, want NotFound", err)
	}
}

func TestFindingRepoListsBySiteStatusCodeAndPage(t *testing.T) {
	t.Parallel()

	store := findingStore(t)
	repo := NewFindingRepo(store)
	product, post := "pg1", "pg2"

	findings := []finding.Finding{
		newFinding(t, "s1", &product, finding.RenderUnreachable, finding.SubjectPage(product), `{}`, findingStamp),
		newFinding(t, "s1", &post, finding.RenderUnreachable, finding.SubjectPage(post), `{}`, findingStamp.Add(time.Minute)),
		newFinding(t, "s1", nil, finding.PluginOutdated, finding.SubjectSite, `{"version":"1.3.0"}`, findingStamp.Add(2*time.Minute)),
		newFinding(t, "s2", nil, finding.PluginOutdated, finding.SubjectSite, `{"version":"1.2.0"}`, findingStamp),
	}
	for i := range findings {
		if err := repo.Upsert(t.Context(), findings[i]); err != nil {
			t.Fatalf("Upsert %s: %v", findings[i].SubjectKey, err)
		}
	}
	if err := repo.Clear(t.Context(), "s1", finding.RenderUnreachable, finding.SubjectPage(post), finding.ResolvedByObservation, finding.ResolutionObservedClear, findingStamp.Add(time.Hour)); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	cleared := findings[1]
	cleared.Status, cleared.Resolution, cleared.ResolvedBy, cleared.UpdatedAt = finding.Resolved, finding.ResolutionObservedClear, finding.ResolvedByObservation, findingStamp.Add(time.Hour)

	for _, check := range []struct {
		why  string
		list func() ([]finding.Finding, error)
		want []finding.Finding
	}{
		{why: "the site's open findings of a code", list: func() ([]finding.Finding, error) {
			return repo.OpenBySite(t.Context(), "s1", finding.RenderUnreachable)
		}, want: []finding.Finding{findings[0]}},
		{why: "the site's open findings", list: func() ([]finding.Finding, error) {
			return repo.ListBySite(t.Context(), "s1", finding.Open)
		}, want: []finding.Finding{findings[0], findings[2]}},
		{why: "the site's resolved findings", list: func() ([]finding.Finding, error) {
			return repo.ListBySite(t.Context(), "s1", finding.Resolved)
		}, want: []finding.Finding{cleared}},
		{why: "every finding of the site", list: func() ([]finding.Finding, error) {
			return repo.ListBySite(t.Context(), "s1", "")
		}, want: []finding.Finding{findings[0], cleared, findings[2]}},
		{why: "the page's findings", list: func() ([]finding.Finding, error) {
			return repo.ListForPage(t.Context(), post)
		}, want: []finding.Finding{cleared}},
	} {
		got, err := check.list()
		if err != nil || !reflect.DeepEqual(got, check.want) {
			t.Fatalf("%s = %+v, %v\nwant %+v", check.why, got, err, check.want)
		}
	}
	if _, err := repo.ListBySite(t.Context(), "s1", "closed"); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a status the record does not name code = %q, want INVALID", errors.CodeOf(err))
	}

	if _, err := store.writer.ExecContext(t.Context(), `DELETE FROM pages WHERE id = ?`, product); err != nil {
		t.Fatalf("delete the page: %v", err)
	}
	if got, err := repo.ListForPage(t.Context(), product); err != nil || len(got) != 0 {
		t.Fatalf("the page's findings after it is deleted = %+v, %v", got, err)
	}
	if _, err := store.writer.ExecContext(t.Context(), `DELETE FROM sites WHERE id = 's1'`); err != nil {
		t.Fatalf("delete the site: %v", err)
	}
	if got, err := repo.ListBySite(t.Context(), "s1", ""); err != nil || len(got) != 0 {
		t.Fatalf("the site's findings after it is deleted = %+v, %v", got, err)
	}
	if got, err := repo.ListBySite(t.Context(), "s2", ""); err != nil || !reflect.DeepEqual(got, findings[3:]) {
		t.Fatalf("the other site's findings = %+v, %v", got, err)
	}
}
