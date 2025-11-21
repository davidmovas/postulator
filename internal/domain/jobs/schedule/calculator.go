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

func (c *Calculator) CalculateNextRun(job *entities.Job, lastRun *time.Time) (baseTime time.Time, withJitter time.Time, err error) {
	if job == nil || job.Schedule == nil {
		return time.Time{}, time.Time{}, nil
	}
	if job.Status != entities.JobStatusActive {
		return time.Time{}, time.Time{}, nil
	}
	if job.Schedule.Type == entities.ScheduleManual {
		return time.Time{}, time.Time{}, nil
	}

	now := time.Now()

	switch job.Schedule.Type {
	case entities.ScheduleOnce:
		baseTime = c.calculateOnce(job.Schedule.Config)
	case entities.ScheduleInterval:
		baseTime = c.calculateInterval(job.Schedule.Config, now, lastRun)
	case entities.ScheduleDaily:
		baseTime = c.calculateDaily(job.Schedule.Config, now)
	default:
		return time.Time{}, time.Time{}, nil
	}

	withJitter = baseTime
	if job.JitterEnabled && job.JitterMinutes > 0 {
		withJitter = c.applyJitter(baseTime, job.JitterMinutes, now)
	}

	return baseTime, withJitter, nil
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
	} else {
		if lastRun != nil && !lastRun.IsZero() {
			anchor = *lastRun
		} else {
			anchor = now
		}
	}

	anchor = time.Date(anchor.Year(), anchor.Month(), anchor.Day(),
		anchor.Hour(), anchor.Minute(), anchor.Second(), 0, now.Location())

	switch cfg.Unit {
	case "hours", "hour":
		return c.calculateHourlyInterval(cfg.Value, anchor, now)
	case "days", "day":
		return c.calculateDailyInterval(cfg.Value, anchor, now)
	case "weeks", "week":
		return c.calculateWeeklyInterval(cfg.Value, anchor, now)
	case "months", "month":
		return c.calculateMonthlyInterval(cfg.Value, anchor, now)
	default:
		return c.calculateDailyInterval(1, anchor, now)
	}
}

func (c *Calculator) calculateHourlyInterval(hours int, anchor, now time.Time) time.Time {
	period := time.Duration(hours) * time.Hour

	if anchor.After(now) {
		return anchor
	}

	elapsed := now.Sub(anchor)
	periodsPassed := elapsed / period

	nextRun := anchor.Add((periodsPassed + 1) * period)

	minNextRun := now.Add(30 * time.Second)
	if nextRun.Before(minNextRun) {
		nextRun = anchor.Add((periodsPassed + 2) * period)
	}

	return nextRun
}

func (c *Calculator) calculateDailyInterval(days int, anchor, now time.Time) time.Time {
	if anchor.After(now) {
		return anchor
	}

	daysSince := int(now.Sub(anchor).Hours() / 24)
	periodsComplete := daysSince / days

	nextRun := anchor.AddDate(0, 0, (periodsComplete+1)*days)

	minNextRun := now.Add(30 * time.Second)
	if nextRun.Before(minNextRun) {
		nextRun = anchor.AddDate(0, 0, (periodsComplete+2)*days)
	}

	return nextRun
}

func (c *Calculator) calculateWeeklyInterval(weeks int, anchor, now time.Time) time.Time {
	days := weeks * 7
	return c.calculateDailyInterval(days, anchor, now)
}

func (c *Calculator) calculateMonthlyInterval(months int, anchor, now time.Time) time.Time {
	if anchor.After(now) {
		return anchor
	}

	monthsDiff := (now.Year()-anchor.Year())*12 + int(now.Month()) - int(anchor.Month())

	if monthsDiff < 0 {
		return anchor
	}

	periodsComplete := monthsDiff / months

	nextRun := anchor.AddDate(0, (periodsComplete+1)*months, 0)

	minNextRun := now.Add(30 * time.Second)
	if nextRun.Before(minNextRun) {
		nextRun = anchor.AddDate(0, (periodsComplete+2)*months, 0)
	}

	return nextRun
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

	candidate := time.Date(now.Year(), now.Month(), now.Day(),
		cfg.Hour, cfg.Minute, 0, 0, now.Location())

	if !candidate.After(now) {
		candidate = candidate.Add(24 * time.Hour)
	}

	for i := 0; i < 7; i++ {
		if len(allowedDays) == 0 || allowedDays[candidate.Weekday()] {
			return candidate
		}
		candidate = candidate.Add(24 * time.Hour)
	}

	return time.Date(now.Year(), now.Month(), now.Day(),
		cfg.Hour, cfg.Minute, 0, 0, now.Location()).Add(24 * time.Hour)
}

func (c *Calculator) applyJitter(baseTime time.Time, jitterMinutes int, now time.Time) time.Time {
	if jitterMinutes <= 0 {
		return baseTime
	}

	jitterOffset := rand.Intn(jitterMinutes*2+1) - jitterMinutes
	jittered := baseTime.Add(time.Duration(jitterOffset) * time.Minute)

	minAllowedTime := now.Add(1 * time.Minute)

	if jittered.Before(minAllowedTime) {
		return baseTime.Add(1 * time.Minute)
	}

	return jittered
}
