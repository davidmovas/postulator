package schedule

import (
	"encoding/json"
	"math"
	"math/rand"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

type Calculator struct{}

func NewCalculator() *Calculator {
	return &Calculator{}
}

func (c *Calculator) CalculateNextRun(job *entities.Job, lastRun *time.Time) (time.Time, error) {
	if job == nil || job.Schedule == nil {
		return time.Time{}, nil
	}
	if job.Status != entities.JobStatusActive {
		return time.Time{}, nil
	}
	if job.Schedule.Type == entities.ScheduleManual {
		return time.Time{}, nil
	}

	now := time.Now()
	var baseCandidate time.Time

	switch job.Schedule.Type {
	case entities.ScheduleOnce:
		baseCandidate = c.calculateOnce(job.Schedule.Config)
	case entities.ScheduleInterval:
		baseCandidate = c.calculateInterval(job.Schedule.Config, now, lastRun)
	case entities.ScheduleDaily:
		baseCandidate = c.calculateDaily(job.Schedule.Config, now)
	default:
		return time.Time{}, nil
	}

	if job.JitterEnabled && job.JitterMinutes > 0 {
		baseCandidate = c.applyJitter(baseCandidate, job.JitterMinutes)
	}

	return baseCandidate, nil
}

func (c *Calculator) calculateOnce(config json.RawMessage) time.Time {
	var cfg entities.OnceSchedule
	if err := json.Unmarshal(config, &cfg); err != nil {
		return time.Time{}
	}
	return cfg.ExecuteAt
}

func (c *Calculator) calculateInterval(config json.RawMessage, now time.Time, lastRun *time.Time) time.Time {
	var cfg entities.IntervalSchedule
	if err := json.Unmarshal(config, &cfg); err != nil {
		return time.Time{}
	}

	if cfg.Value <= 0 {
		cfg.Value = 1
	}

	var anchor time.Time
	if cfg.StartAt != nil && !cfg.StartAt.IsZero() {
		anchor = cfg.StartAt.In(now.Location())
	} else if lastRun != nil && !lastRun.IsZero() {
		anchor = lastRun.In(now.Location())
	} else {
		anchor = now
	}

	anchor = time.Date(anchor.Year(), anchor.Month(), anchor.Day(), anchor.Hour(), anchor.Minute(), anchor.Second(), anchor.Nanosecond(), now.Location())

	switch cfg.Unit {
	case "hours", "hour":
		period := time.Duration(cfg.Value) * time.Hour
		if anchor.After(now) {
			return anchor
		}
		diff := now.Sub(anchor)
		k := int64(diff/period) + 1
		candidate := anchor.Add(time.Duration(k) * period)
		if !candidate.After(now) {
			candidate = candidate.Add(period)
		}
		return candidate

	case "days", "day":
		if anchor.After(now) {
			return anchor
		}
		periodDays := cfg.Value
		daysDiff := int(now.Sub(anchor) / (24 * time.Hour))
		k := daysDiff/periodDays + 1
		candidate := anchor.AddDate(0, 0, k*periodDays)
		for !candidate.After(now) {
			candidate = candidate.AddDate(0, 0, periodDays)
		}
		return candidate

	case "weeks", "week":
		if anchor.After(now) {
			return anchor
		}
		periodDays := cfg.Value * 7
		daysDiff := int(now.Sub(anchor) / (24 * time.Hour))
		k := daysDiff/periodDays + 1
		candidate := anchor.AddDate(0, 0, k*periodDays)
		for !candidate.After(now) {
			candidate = candidate.AddDate(0, 0, periodDays)
		}
		return candidate

	case "months", "month":
		if anchor.After(now) {
			return anchor
		}
		candidate := anchor
		monthsDiff := (now.Year()-anchor.Year())*12 + int(now.Month()) - int(anchor.Month())
		approxSteps := int(math.Floor(float64(monthsDiff) / float64(cfg.Value)))
		if approxSteps < 0 {
			approxSteps = 0
		}
		if approxSteps > 0 {
			candidate = candidate.AddDate(0, approxSteps*cfg.Value, 0)
		}
		for !candidate.After(now) {
			candidate = candidate.AddDate(0, cfg.Value, 0)
		}
		return candidate

	default:
		// default to 1 day period
		if anchor.After(now) {
			return anchor
		}
		candidate := anchor.Add(24 * time.Hour)
		for !candidate.After(now) {
			candidate = candidate.Add(24 * time.Hour)
		}
		return candidate
	}
}

func (c *Calculator) calculateDaily(config json.RawMessage, now time.Time) time.Time {
	var cfg entities.DailySchedule
	if err := json.Unmarshal(config, &cfg); err != nil {
		return time.Time{}
	}

	allowedDays := make(map[time.Weekday]bool)
	for _, day := range cfg.Weekdays {
		weekday := time.Weekday(day % 7)
		allowedDays[weekday] = true
	}

	candidate := time.Date(now.Year(), now.Month(), now.Day(), cfg.Hour, cfg.Minute, 0, 0, now.Location())

	if !candidate.After(now) {
		candidate = candidate.Add(24 * time.Hour)
	}

	for i := 0; i < 8; i++ {
		if len(allowedDays) == 0 || allowedDays[candidate.Weekday()] {
			return candidate
		}
		candidate = candidate.Add(24 * time.Hour)
	}

	return time.Date(now.Year(), now.Month(), now.Day(), cfg.Hour, cfg.Minute, 0, 0, now.Location()).Add(24 * time.Hour)
}

func (c *Calculator) applyJitter(t time.Time, jitterMinutes int) time.Time {
	if jitterMinutes <= 0 {
		return t
	}

	j := rand.Intn(jitterMinutes*2+1) - jitterMinutes
	jittered := t.Add(time.Duration(j) * time.Minute)

	if jittered.Before(time.Now()) {
		jittered = time.Now().Add(time.Minute * 2)
	}

	return jittered
}
