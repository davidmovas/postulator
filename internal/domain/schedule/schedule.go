package schedule

import (
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	MinInterval  = time.Minute
	MaxInterval  = 30 * 24 * time.Hour
	MaxTargets   = 500
	MaxNameRunes = 120
)

type TargetQuery struct {
	EntityID *string
	Status   string
	Limit    int
}

type Schedule struct {
	ID          string
	SiteID      string
	Name        string
	Cron        string
	Interval    *time.Duration
	Query       TargetQuery
	TemplateID  string
	Recipe      []template.StepSpec
	PublishMode run.PublishMode
	Budget      run.Budget
	Enabled     bool
	NextRunAt   *time.Time
	LastRunID   *string
	CreatedBy   kctx.Actor
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func invalid(message, field string) *errors.Error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}

func New(s Schedule) (Schedule, error) {
	s.Name = strings.Join(strings.Fields(s.Name), " ")
	s.Cron = strings.TrimSpace(s.Cron)
	if s.PublishMode == "" {
		s.PublishMode = run.PublishDraft
	}
	if s.CreatedBy == "" {
		s.CreatedBy = kctx.ActorUser
	}
	if s.Query.Limit <= 0 {
		s.Query.Limit = MaxTargets
	}

	switch {
	case s.ID == "":
		return Schedule{}, invalid("schedule id must not be empty", "id")
	case s.SiteID == "":
		return Schedule{}, invalid("a schedule belongs to a site", "siteId")
	case s.Name == "":
		return Schedule{}, invalid("a schedule needs a name", "name")
	case len([]rune(s.Name)) > MaxNameRunes:
		return Schedule{}, invalid("the schedule name is too long", "name")
	case s.Cron == "" && s.Interval == nil:
		return Schedule{}, invalid("a schedule needs a cron expression or an interval", "cron")
	case s.Cron != "" && s.Interval != nil:
		return Schedule{}, invalid("a schedule runs on a cron expression or on an interval, not on both", "cron")
	case s.Interval != nil && (*s.Interval < MinInterval || *s.Interval > MaxInterval):
		return Schedule{}, invalid("the interval must be between one minute and thirty days", "interval")
	case !s.PublishMode.Valid():
		return Schedule{}, invalid("publish mode is not recognized", "publishMode")
	case s.Query.Limit > MaxTargets:
		return Schedule{}, invalid("a schedule may not target more than 500 pages at once", "limit")
	case s.Query.EntityID != nil && *s.Query.EntityID == "":
		return Schedule{}, invalid("the target entity must not be empty when set", "entityId")
	case s.Budget.MaxUSD < 0 || s.Budget.MaxTokens < 0:
		return Schedule{}, invalid("a budget must not be negative", "budget")
	case s.LastRunID != nil && *s.LastRunID == "":
		return Schedule{}, invalid("the last run must not be empty when set", "lastRunId")
	case s.CreatedAt.IsZero():
		return Schedule{}, invalid("a schedule needs a creation timestamp", "createdAt")
	}

	if s.Cron != "" {
		if _, err := parse(s.Cron); err != nil {
			return Schedule{}, err
		}
	}
	return s, nil
}

func (s Schedule) NextAfter(now time.Time) (time.Time, error) {
	if s.Interval != nil {
		return now.UTC().Add(*s.Interval), nil
	}

	parsed, err := parse(s.Cron)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.Next(now.UTC()).UTC(), nil
}

func parse(expression string) (cron.Schedule, error) {
	parsed, err := cron.ParseStandard(expression)
	if err != nil {
		return nil, errors.New(errors.Invalid, "the cron expression could not be read").
			WithDetail("field", "cron").WithInternal(err)
	}
	return parsed, nil
}

type Query struct {
	SiteID  string
	Enabled *bool
	Desc    bool
}
