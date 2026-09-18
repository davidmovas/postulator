package schedules_test

import (
	"context"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/schedules"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	domainrun "github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

type startedRun struct {
	request runs.StartRequest
	actor   kctx.Actor
}

type runStub struct {
	runs    *sqlite.RunRepo
	started []startedRun
	status  domainrun.Status
	err     error
	missing bool
}

func (r *runStub) Start(ctx context.Context, req runs.StartRequest) (runs.StartResponse, error) {
	if r.err != nil {
		return runs.StartResponse{}, r.err
	}

	actor, _ := kctx.ActorFrom(ctx)
	r.started = append(r.started, startedRun{request: req, actor: actor})

	recipe := req.Recipe
	if len(recipe) == 0 {
		recipe = []template.StepSpec{{Name: "report", Enabled: true}}
	}
	record, err := domainrun.NewRun(domainrun.Run{
		ID: id.New(), SiteID: req.SiteID, Kind: domainrun.KindGenerate, Targets: req.PageIDs,
		Recipe: recipe, CreatedBy: actor, CreatedAt: sqlitetest.Stamp,
		DeadlineAt: sqlitetest.Stamp.Add(time.Hour),
	})
	if err != nil {
		return runs.StartResponse{}, err
	}
	if insertErr := r.runs.Insert(ctx, record); insertErr != nil {
		return runs.StartResponse{}, insertErr
	}
	return runs.StartResponse{RunID: record.ID}, nil
}

func (r *runStub) Get(context.Context, string) (domainrun.Run, error) {
	if r.missing {
		return domainrun.Run{}, errors.New(errors.NotFound, "run not found")
	}
	return domainrun.Run{Status: r.status}, nil
}

type fixture struct {
	service  *schedules.Service
	store    *sqlite.Store
	runner   *runStub
	clock    *clock.Fake
	siteID   string
	entityID string
	pageIDs  []string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	entity := sqlitetest.Entity(t, store, owner.ID, "Coffee")
	fake := clock.NewFake(sqlitetest.Stamp)
	runner := &runStub{runs: sqlite.NewRunRepo(store), status: domainrun.StatusCompleted}

	pageRepo := sqlite.NewPageRepo(store)
	ids := make([]string, 0, 3)
	for i, path := range []string{"/coffee/", "/coffee/espresso/", "/coffee/filter/"} {
		page := pagemap.Page{
			ID: id.New(), SiteID: owner.ID, Path: path, Slug: pagemap.Slug(path), WPType: pagemap.WPPage,
			Status: pagemap.StatusPlanned, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
		}
		if i == 0 {
			page.Status = pagemap.StatusPublished
		}
		if i == 1 {
			page.EntityID = &entity.ID
		}
		if err := pageRepo.Insert(t.Context(), page); err != nil {
			t.Fatalf("insert the page: %v", err)
		}
		ids = append(ids, page.ID)
	}

	return &fixture{
		service: schedules.New(schedules.Deps{
			Schedules: sqlite.NewScheduleRepo(store), Pages: pageRepo, Sites: sqlite.NewSiteRepo(store),
			Runs: runner, RunReader: runner, Clock: fake,
		}),
		store: store, runner: runner, clock: fake, siteID: owner.ID, entityID: entity.ID, pageIDs: ids,
	}
}

func (f *fixture) create(t *testing.T, mutate func(*schedules.CreateRequest)) schedules.Schedule {
	t.Helper()

	request := schedules.CreateRequest{
		SiteID: f.siteID, Name: "nightly refresh", Cron: "0 3 * * *", Status: string(pagemap.StatusPlanned),
		Limit: 10, Steps: []string{"resolve_context", "generate_body"}, Enabled: true,
	}
	if mutate != nil {
		mutate(&request)
	}

	created, err := f.service.Create(t.Context(), request)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	return created.Schedule
}

func TestCreateArmsTheNextRun(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	created := f.create(t, nil)

	if created.NextRunAt.Std().IsZero() || created.Steps[0] != "resolve_context" {
		t.Fatalf("the schedule is %+v", created)
	}
	if created.PublishMode != string(domainrun.PublishDraft) || created.CreatedBy != string(kctx.ActorUser) {
		t.Fatalf("the schedule is %+v", created)
	}

	listed, err := f.service.List(t.Context(), schedules.ListRequest{SiteID: f.siteID})
	if err != nil || len(listed.Items) != 1 {
		t.Fatalf("List = %+v, %v", listed, err)
	}

	read, err := f.service.Get(t.Context(), schedules.GetRequest{ID: created.ID})
	if err != nil || read.Schedule.Name != "nightly refresh" {
		t.Fatalf("Get = %+v, %v", read, err)
	}
}

func TestCreateRefusesWhatItCannotSchedule(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	cases := []struct {
		name    string
		request schedules.CreateRequest
		want    errors.Code
	}{
		{name: "no site", request: schedules.CreateRequest{Name: "n", Cron: "@daily"}, want: errors.Invalid},
		{
			name:    "unknown site",
			request: schedules.CreateRequest{SiteID: id.New(), Name: "n", Cron: "@daily"},
			want:    errors.NotFound,
		},
		{
			name:    "an unknown target status",
			request: schedules.CreateRequest{SiteID: f.siteID, Name: "n", Cron: "@daily", Status: "somewhere"},
			want:    errors.Invalid,
		},
		{
			name:    "an unreadable cron expression",
			request: schedules.CreateRequest{SiteID: f.siteID, Name: "n", Cron: "every tuesday"},
			want:    errors.Invalid,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := f.service.Create(t.Context(), tc.request); !errors.IsCode(err, tc.want) {
				t.Fatalf("Create = %v, want %s", err, tc.want)
			}
		})
	}
}

func TestUpdateSwapsTheTiming(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	created := f.create(t, nil)

	minutes := 30
	updated, err := f.service.Update(t.Context(), schedules.UpdateRequest{
		ID: created.ID, IntervalMinutes: &minutes,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Schedule.Cron != "" || updated.Schedule.IntervalMinutes != 30 {
		t.Fatalf("the schedule is %+v", updated.Schedule)
	}

	expression := "0 4 * * *"
	back, err := f.service.Update(t.Context(), schedules.UpdateRequest{ID: created.ID, Cron: &expression})
	if err != nil || back.Schedule.IntervalMinutes != 0 || back.Schedule.Cron != expression {
		t.Fatalf("Update = %+v, %v", back.Schedule, err)
	}

	name, status, limit := "morning", string(pagemap.StatusPublished), 3
	templateID, mode, usd, tokens := "t1", string(domainrun.PublishLive), 2.5, 1000
	entity := f.entityID
	full, err := f.service.Update(t.Context(), schedules.UpdateRequest{
		ID: created.ID, Name: &name, EntityID: &entity, Status: &status, Limit: &limit,
		TemplateID: &templateID, Steps: []string{"report"}, PublishMode: &mode, MaxUSD: &usd, MaxTokens: &tokens,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if full.Schedule.Name != "morning" || full.Schedule.Limit != 3 || full.Schedule.MaxUSD != 2.5 {
		t.Fatalf("the schedule is %+v", full.Schedule)
	}

	unknown := "nowhere"
	if _, err = f.service.Update(t.Context(), schedules.UpdateRequest{
		ID: created.ID, Status: &unknown,
	}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Update to an unknown status = %v", err)
	}
	if _, err = f.service.Update(t.Context(), schedules.UpdateRequest{ID: id.New()}); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Update of an unknown schedule = %v", err)
	}
}

func TestDisableDisarmsAndEnableArmsAgain(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	created := f.create(t, nil)

	off, err := f.service.Disable(t.Context(), schedules.DisableRequest{ID: created.ID})
	if err != nil || off.Schedule.Enabled || !off.Schedule.NextRunAt.Std().IsZero() {
		t.Fatalf("Disable = %+v, %v", off.Schedule, err)
	}

	on, err := f.service.Enable(t.Context(), schedules.EnableRequest{ID: created.ID})
	if err != nil || !on.Schedule.Enabled || on.Schedule.NextRunAt.Std().IsZero() {
		t.Fatalf("Enable = %+v, %v", on.Schedule, err)
	}

	if _, err = f.service.Delete(t.Context(), schedules.DeleteRequest{ID: created.ID}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err = f.service.Get(t.Context(), schedules.GetRequest{ID: created.ID}); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Get after the delete = %v", err)
	}
	if _, err = f.service.Delete(t.Context(), schedules.DeleteRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Delete without an id = %v", err)
	}
}

func TestRunNowStartsTheTargetsTheQueryNames(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	created := f.create(t, nil)

	started, err := f.service.RunNow(t.Context(), schedules.RunNowRequest{ID: created.ID})
	if err != nil {
		t.Fatalf("RunNow: %v", err)
	}
	if started.RunID == "" || started.Targets != 2 {
		t.Fatalf("RunNow = %+v, want the two planned pages", started)
	}
	if len(f.runner.started) != 1 || f.runner.started[0].actor != kctx.ActorSchedule {
		t.Fatalf("the run was started as %+v", f.runner.started)
	}
	if f.runner.started[0].request.SiteID != f.siteID || len(f.runner.started[0].request.Recipe) != 2 {
		t.Fatalf("the start request is %+v", f.runner.started[0].request)
	}

	read, err := f.service.Get(t.Context(), schedules.GetRequest{ID: created.ID})
	if err != nil || read.Schedule.LastRunID == nil || *read.Schedule.LastRunID != started.RunID {
		t.Fatalf("the schedule is %+v, %v", read.Schedule, err)
	}
}

func TestRunNowFiltersByEntity(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	created := f.create(t, func(req *schedules.CreateRequest) {
		req.EntityID = f.entityID
		req.Status = ""
	})

	started, err := f.service.RunNow(t.Context(), schedules.RunNowRequest{ID: created.ID})
	if err != nil || started.Targets != 1 {
		t.Fatalf("RunNow = %+v, %v", started, err)
	}
	if f.runner.started[0].request.PageIDs[0] != f.pageIDs[1] {
		t.Fatalf("the targets are %v", f.runner.started[0].request.PageIDs)
	}
}

func TestATickSkipsAScheduleWhosePreviousRunIsStillGoing(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	created := f.create(t, nil)

	if _, err := f.service.RunNow(t.Context(), schedules.RunNowRequest{ID: created.ID}); err != nil {
		t.Fatalf("RunNow: %v", err)
	}
	f.runner.status = domainrun.StatusRunning

	skipped, err := f.service.RunNow(t.Context(), schedules.RunNowRequest{ID: created.ID})
	if err != nil {
		t.Fatalf("RunNow: %v", err)
	}
	if skipped.RunID != "" || skipped.Skipped != schedules.SkippedStillRunning {
		t.Fatalf("RunNow = %+v, want the skip", skipped)
	}
	if len(f.runner.started) != 1 {
		t.Fatalf("the runner started %d runs", len(f.runner.started))
	}
}

func TestATickStartsWhatIsDueAndRearmsIt(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	created := f.create(t, func(req *schedules.CreateRequest) {
		req.Cron = ""
		req.IntervalMinutes = 60
	})

	idle, err := f.service.Tick(t.Context())
	if err != nil || len(idle.Started) != 0 {
		t.Fatalf("Tick before the schedule is due = %+v, %v", idle, err)
	}

	f.clock.Advance(90 * time.Minute)
	ticked, err := f.service.Tick(t.Context())
	if err != nil || len(ticked.Started) != 1 {
		t.Fatalf("Tick = %+v, %v", ticked, err)
	}

	read, err := f.service.Get(t.Context(), schedules.GetRequest{ID: created.ID})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if read.Schedule.NextRunAt.Std().IsZero() {
		t.Fatalf("the schedule was not rearmed: %+v", read.Schedule)
	}

	again, err := f.service.Tick(t.Context())
	if err != nil || len(again.Started) != 0 {
		t.Fatalf("a rearmed schedule must not fire twice: %+v, %v", again, err)
	}
}

func TestATickSkipsASchedulWhoseQueryMatchesNothing(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	created := f.create(t, func(req *schedules.CreateRequest) { req.Status = string(pagemap.StatusArchived) })

	skipped, err := f.service.RunNow(t.Context(), schedules.RunNowRequest{ID: created.ID})
	if err != nil || skipped.RunID != "" || skipped.Skipped == "" {
		t.Fatalf("RunNow = %+v, %v", skipped, err)
	}
	if len(f.runner.started) != 0 {
		t.Fatal("a schedule with no targets must not start a run")
	}
}

func TestRunNowCarriesTheStartFailureUp(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	created := f.create(t, nil)
	f.runner.err = errors.New(errors.External, "the engine is down")

	if _, err := f.service.RunNow(t.Context(), schedules.RunNowRequest{ID: created.ID}); !errors.IsCode(err, errors.External) {
		t.Fatalf("RunNow = %v", err)
	}
	if _, err := f.service.RunNow(t.Context(), schedules.RunNowRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("RunNow without an id = %v", err)
	}
}

func TestAMissingPreviousRunDoesNotBlockTheNextOne(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	created := f.create(t, nil)

	if _, err := f.service.RunNow(t.Context(), schedules.RunNowRequest{ID: created.ID}); err != nil {
		t.Fatalf("RunNow: %v", err)
	}
	f.runner.missing = true

	started, err := f.service.RunNow(t.Context(), schedules.RunNowRequest{ID: created.ID})
	if err != nil || started.RunID == "" {
		t.Fatalf("RunNow after a purged run = %+v, %v", started, err)
	}
}
