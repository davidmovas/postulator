package sites_test

import (
	stderrors "errors"
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/domain/template"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func observedSite(t *testing.T, h harness) site.Site {
	t.Helper()
	created, err := h.service.Create(t.Context(), sites.CreateRequest{
		Name: "Shop", BaseURL: "https://shop.example.com", Username: "editor", Password: "abcd efgh",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	walked := sqlitetest.Stamp
	if err = sqlite.NewSiteRepo(h.store).UpdateObservation(t.Context(), created.Site.ID, site.TechnicalPages{Front: 2, Cart: 8},
		site.ObservationState{LastStartedAt: &walked, LastFinishedAt: &walked, LastFullWalkAt: &walked, LastModifiedSeen: &walked, SinceFullWalk: 3},
		walked); err != nil {
		t.Fatalf("UpdateObservation: %v", err)
	}
	stored, err := sqlite.NewSiteRepo(h.store).Get(t.Context(), created.Site.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	return stored
}

func plantRun(t *testing.T, h harness, siteID string, status run.Status) string {
	t.Helper()
	record := run.Run{
		ID: id.New(), SiteID: siteID, Kind: run.KindGenerate, Status: status, Targets: []string{},
		Recipe:     []template.StepSpec{{Name: "generate_body", Enabled: true}},
		TemplateID: id.New(), TemplateVersion: 1, PublishMode: run.PublishDraft,
		CreatedBy: kctx.ActorUser, DeadlineAt: sqlitetest.Stamp.Add(time.Hour), CreatedAt: sqlitetest.Stamp,
	}
	if err := sqlite.NewRunRepo(h.store).Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the run: %v", err)
	}
	return record.ID
}

func detailOf(t *testing.T, err error, key string) any {
	t.Helper()
	kernel, ok := stderrors.AsType[*errors.Error](err)
	if !ok {
		t.Fatalf("error %v is not a kernel error", err)
	}
	return kernel.Details[key]
}

func eventTypes(recorded []events.Type) []events.Type {
	out := slices.Clone(recorded)
	slices.Sort(out)
	return out
}

func TestClearDataRemovesTheChosenGroupsAndKeepsTheSetup(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		req      sites.ClearDataRequest
		held     sites.DataHeld
		observed bool
		events   []events.Type
	}{
		{name: "the model takes the history and the observation", req: sites.ClearDataRequest{Model: true},
			held: sites.DataHeld{Entities: 1}, observed: false,
			events: []events.Type{events.GraphChanged, events.PagesChanged, events.SitesChanged}},
		{name: "the graph keeps the pages", req: sites.ClearDataRequest{Graph: true},
			held: sites.DataHeld{Pages: 1, Runs: 1}, observed: true,
			events: []events.Type{events.GraphChanged, events.PagesChanged, events.SitesChanged}},
		{name: "the history keeps the model", req: sites.ClearDataRequest{History: true},
			held: sites.DataHeld{Pages: 1, Entities: 1}, observed: true,
			events: []events.Type{events.SitesChanged}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			stored := observedSite(t, h)
			sqlitetest.Entity(t, h.store, stored.ID, "Shoes")
			sqlitetest.Page(t, h.store, stored.ID, "/shoes/")
			plantRun(t, h, stored.ID, run.StatusCompleted)
			typesBefore, err := sqlite.NewPageTypeRepo(h.store).ListBySite(t.Context(), stored.ID)
			if err != nil {
				t.Fatalf("list the types: %v", err)
			}
			h.events.Reset()

			req := tc.req
			req.ID = stored.ID
			answered, err := h.service.ClearData(t.Context(), req)
			if err != nil {
				t.Fatalf("ClearData = %v", err)
			}
			if answered.Held != tc.held {
				t.Errorf("held after = %+v, want %+v", answered.Held, tc.held)
			}
			after, err := sqlite.NewSiteRepo(h.store).Get(t.Context(), stored.ID)
			if err != nil {
				t.Fatalf("the site itself must stay: %v", err)
			}
			if observed := after.Settings.Observation.LastFullWalkAt != nil; observed != tc.observed {
				t.Errorf("observation kept = %v, want %v (%+v)", observed, tc.observed, after.Settings.Observation)
			}
			if !tc.observed && after.Settings.Technical != (site.TechnicalPages{}) {
				t.Errorf("technical pages = %+v, want them learned again", after.Settings.Technical)
			}
			if after.Name != stored.Name || after.BaseURL != stored.BaseURL || after.Username != stored.Username {
				t.Errorf("the connection moved: %+v", after)
			}
			if _, err = h.secrets.Get(t.Context(), stored.SecretRef); err != nil {
				t.Errorf("the password must stay: %v", err)
			}
			typesAfter, err := sqlite.NewPageTypeRepo(h.store).ListBySite(t.Context(), stored.ID)
			if err != nil || len(typesAfter) != len(typesBefore) {
				t.Errorf("page types after = %d, %v; want the %d the site had", len(typesAfter), err, len(typesBefore))
			}
			recorded := h.events.Events()
			published := make([]events.Type, 0, len(recorded))
			for _, event := range recorded {
				published = append(published, event.Type)
			}
			if got := eventTypes(published); !slices.Equal(got, tc.events) {
				t.Errorf("published %v, want %v", got, tc.events)
			}
		})
	}
}

func TestDataCountsSaysWhatEachGroupHolds(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	stored := observedSite(t, h)
	sqlitetest.Entity(t, h.store, stored.ID, "Shoes")
	sqlitetest.Page(t, h.store, stored.ID, "/shoes/")
	plantRun(t, h, stored.ID, run.StatusPaused)
	running := plantRun(t, h, stored.ID, run.StatusRunning)

	answered, err := h.service.DataCounts(t.Context(), sites.DataCountsRequest{ID: stored.ID})
	if err != nil {
		t.Fatalf("DataCounts = %v", err)
	}
	want := sites.DataHeld{Pages: 1, Entities: 1, Runs: 2, Unfinished: 1}
	if answered.Held != want {
		t.Errorf("held = %+v, want %+v", answered.Held, want)
	}
	if !slices.Equal(answered.Busy, []string{running}) {
		t.Errorf("busy = %v, want the running run %s", answered.Busy, running)
	}
	if _, err = h.service.DataCounts(t.Context(), sites.DataCountsRequest{ID: id.New()}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("DataCounts of an unknown site = %v, want NOT_FOUND", err)
	}
}

func TestClearAndDeleteRefuseWhatTheyCannotDo(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	stored := observedSite(t, h)
	sqlitetest.Page(t, h.store, stored.ID, "/shoes/")
	running := plantRun(t, h, stored.ID, run.StatusRunning)

	cases := []struct {
		name string
		call func() error
		code errors.Code
		busy bool
	}{
		{name: "a clear that names no group", code: errors.Invalid, call: func() error {
			_, err := h.service.ClearData(t.Context(), sites.ClearDataRequest{ID: stored.ID})
			return err
		}},
		{name: "a clear of an unknown site", code: errors.NotFound, call: func() error {
			_, err := h.service.ClearData(t.Context(), sites.ClearDataRequest{ID: id.New(), Graph: true})
			return err
		}},
		{name: "a clear while a run is under way", code: errors.Conflict, busy: true, call: func() error {
			_, err := h.service.ClearData(t.Context(), sites.ClearDataRequest{ID: stored.ID, Model: true})
			return err
		}},
		{name: "a delete while a run is under way", code: errors.Conflict, busy: true, call: func() error {
			_, err := h.service.Delete(t.Context(), sites.DeleteRequest{ID: stored.ID})
			return err
		}},
	}
	for _, tc := range cases {
		err := tc.call()
		if !errors.IsCode(err, tc.code) {
			t.Errorf("%s = %v, want %s", tc.name, err, tc.code)
			continue
		}
		if !tc.busy {
			continue
		}
		if code := detailOf(t, err, "code"); code != sites.CodeSiteBusy {
			t.Errorf("%s: code detail = %v, want %s", tc.name, code, sites.CodeSiteBusy)
		}
		if runs, ok := detailOf(t, err, "runIds").([]string); !ok || !slices.Equal(runs, []string{running}) {
			t.Errorf("%s: runIds = %v, want the running run", tc.name, detailOf(t, err, "runIds"))
		}
	}
	if pages, err := sqlite.NewPageRepo(h.store).ListBySite(t.Context(), stored.ID); err != nil || len(pages) != 1 {
		t.Errorf("a refused clear or delete removed something: %d pages, %v", len(pages), err)
	}
}
