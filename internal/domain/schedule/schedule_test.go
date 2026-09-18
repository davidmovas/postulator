package schedule_test

import (
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/schedule"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

var stamp = time.Date(2026, time.September, 18, 9, 30, 0, 0, time.UTC)

func valid() schedule.Schedule {
	return schedule.Schedule{
		ID: "s1", SiteID: "site-1", Name: "  nightly   refresh ", Cron: "0 3 * * *",
		Query: schedule.TargetQuery{Status: "planned", Limit: 10}, Enabled: true, CreatedAt: stamp,
	}
}

func interval(d time.Duration) *time.Duration {
	return &d
}

func TestNewFillsTheDefaults(t *testing.T) {
	t.Parallel()

	created, err := schedule.New(valid())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if created.Name != "nightly refresh" {
		t.Fatalf("name = %q", created.Name)
	}
	if created.PublishMode != run.PublishDraft || created.CreatedBy != kctx.ActorUser {
		t.Fatalf("schedule = %+v", created)
	}

	bare := valid()
	bare.Query.Limit = 0
	unlimited, err := schedule.New(bare)
	if err != nil || unlimited.Query.Limit != schedule.MaxTargets {
		t.Fatalf("New = %+v, %v", unlimited, err)
	}
}

func TestNewRefusesWhatItCannotSchedule(t *testing.T) {
	t.Parallel()

	empty := ""
	cases := []struct {
		name   string
		mutate func(*schedule.Schedule)
	}{
		{name: "no id", mutate: func(s *schedule.Schedule) { s.ID = "" }},
		{name: "no site", mutate: func(s *schedule.Schedule) { s.SiteID = "" }},
		{name: "no name", mutate: func(s *schedule.Schedule) { s.Name = "   " }},
		{name: "a name that will not fit", mutate: func(s *schedule.Schedule) { s.Name = longName() }},
		{name: "neither cron nor interval", mutate: func(s *schedule.Schedule) { s.Cron = "" }},
		{name: "both cron and interval", mutate: func(s *schedule.Schedule) { s.Interval = interval(time.Hour) }},
		{
			name:   "an interval that is too short",
			mutate: func(s *schedule.Schedule) { s.Cron = ""; s.Interval = interval(time.Second) },
		},
		{
			name:   "an interval that is too long",
			mutate: func(s *schedule.Schedule) { s.Cron = ""; s.Interval = interval(400 * 24 * time.Hour) },
		},
		{name: "an unreadable cron expression", mutate: func(s *schedule.Schedule) { s.Cron = "every other tuesday" }},
		{name: "an unknown publish mode", mutate: func(s *schedule.Schedule) { s.PublishMode = "broadcast" }},
		{name: "too many targets", mutate: func(s *schedule.Schedule) { s.Query.Limit = schedule.MaxTargets + 1 }},
		{name: "an empty target entity", mutate: func(s *schedule.Schedule) { s.Query.EntityID = &empty }},
		{name: "a negative budget", mutate: func(s *schedule.Schedule) { s.Budget = run.Budget{MaxUSD: -1} }},
		{name: "an empty last run", mutate: func(s *schedule.Schedule) { s.LastRunID = &empty }},
		{name: "no timestamp", mutate: func(s *schedule.Schedule) { s.CreatedAt = time.Time{} }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			candidate := valid()
			tc.mutate(&candidate)
			if _, err := schedule.New(candidate); !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("New = %v, want invalid", err)
			}
		})
	}
}

func TestNextAfterFollowsTheCronOrTheInterval(t *testing.T) {
	t.Parallel()

	nightly, err := schedule.New(valid())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	next, err := nightly.NextAfter(stamp)
	if err != nil {
		t.Fatalf("NextAfter: %v", err)
	}
	want := time.Date(2026, time.September, 19, 3, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("NextAfter = %s, want %s", next, want)
	}

	every := valid()
	every.Cron = ""
	every.Interval = interval(2 * time.Hour)
	repeating, err := schedule.New(every)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if next, err = repeating.NextAfter(stamp); err != nil || !next.Equal(stamp.Add(2*time.Hour)) {
		t.Fatalf("NextAfter = %s, %v", next, err)
	}

	broken := schedule.Schedule{Cron: "not a cron"}
	if _, err = broken.NextAfter(stamp); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("NextAfter of an unreadable expression = %v", err)
	}
}

func TestADescriptorIsACronExpressionToo(t *testing.T) {
	t.Parallel()

	daily := valid()
	daily.Cron = "@daily"

	created, err := schedule.New(daily)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	next, err := created.NextAfter(stamp)
	if err != nil {
		t.Fatalf("NextAfter: %v", err)
	}
	if want := time.Date(2026, time.September, 19, 0, 0, 0, 0, time.UTC); !next.Equal(want) {
		t.Fatalf("NextAfter = %s, want %s", next, want)
	}
}

func longName() string {
	name := make([]rune, schedule.MaxNameRunes+1)
	for i := range name {
		name[i] = 'a'
	}
	return string(name)
}
