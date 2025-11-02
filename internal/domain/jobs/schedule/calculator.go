package schedule

import (
	"encoding/json"
	"math/rand"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

type Calculator struct{}

func NewCalculator() *Calculator {
	return &Calculator{}
}

func (c *Calculator) CalculateNextRun(job *entities.Job, lastRun *time.Time) (time.Time, error) {
	if job.Schedule == nil {
		return time.Time{}, nil
	}

	if job.Status != entities.JobStatusActive {
		return time.Time{}, nil
	}

	var nextRun time.Time
	now := time.Now()

	switch job.Schedule.Type {
	case entities.ScheduleManual:
		return time.Time{}, nil
	case entities.ScheduleOnce:
		nextRun = c.calculateOnce(job.Schedule.Config, now)
	case entities.ScheduleInterval:
		nextRun = c.calculateInterval(job.Schedule.Config, now, lastRun)
	case entities.ScheduleDaily:
		nextRun = c.calculateDaily(job.Schedule.Config, now)
	default:
		return time.Time{}, nil
	}

	if job.JitterEnabled && job.JitterMinutes > 0 {
		nextRun = c.applyJitter(nextRun, job.JitterMinutes)
	}

	return nextRun, nil
}

func (c *Calculator) calculateOnce(config json.RawMessage, now time.Time) time.Time {
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

	baseTime := now
	if lastRun != nil && !lastRun.IsZero() {
		baseTime = *lastRun
	}

	switch cfg.Unit {
	case "hours":
		return baseTime.Add(time.Duration(cfg.Value) * time.Hour)
	case "days":
		return baseTime.AddDate(0, 0, cfg.Value)
	case "weeks":
		return baseTime.AddDate(0, 0, cfg.Value*7)
	case "months":
		return baseTime.AddDate(0, cfg.Value, 0)
	default:
		return baseTime.Add(24 * time.Hour)
	}
}

func (c *Calculator) calculateDaily(config json.RawMessage, now time.Time) time.Time {
	var cfg entities.DailySchedule
	if err := json.Unmarshal(config, &cfg); err != nil {
		return time.Time{}
	}

	allowedDays := make(map[time.Weekday]bool)
	for _, day := range cfg.Weekdays {
		weekday := time.Weekday((day) % 7)
		allowedDays[weekday] = true
	}

	candidate := time.Date(now.Year(), now.Month(), now.Day(), cfg.Hour, cfg.Minute, 0, 0, now.Location())

	if candidate.Before(now) || candidate.Equal(now) {
		candidate = candidate.Add(24 * time.Hour)
	}

	for i := 0; i < 8; i++ {
		if allowedDays[candidate.Weekday()] {
			return candidate
		}
		candidate = candidate.Add(24 * time.Hour)
	}

	return now.Add(24 * time.Hour)
}

func (c *Calculator) applyJitter(t time.Time, jitterMinutes int) time.Time {
	jitter := rand.Intn(jitterMinutes*2+1) - jitterMinutes
	return t.Add(time.Duration(jitter) * time.Minute)
}
