package schedule

import (
	"encoding/json"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/pkg/errors"
)

func ValidateSchedule(schedule *entities.Schedule) error {
	if schedule == nil {
		return errors.Validation("schedule is required")
	}

	switch schedule.Type {
	case entities.ScheduleManual:
		return nil
	case entities.ScheduleOnce:
		return validateOnceSchedule(schedule.Config)
	case entities.ScheduleInterval:
		return validateIntervalSchedule(schedule.Config)
	case entities.ScheduleDaily:
		return validateDailySchedule(schedule.Config)
	default:
		return errors.Validation("invalid schedule type")
	}
}

func validateOnceSchedule(config json.RawMessage) error {
	var cfg entities.OnceSchedule
	if err := json.Unmarshal(config, &cfg); err != nil {
		return errors.Validation("invalid once schedule config")
	}

	if cfg.ExecuteAt.IsZero() {
		return errors.Validation("execute_at is required for once schedule")
	}

	if cfg.ExecuteAt.Before(time.Now()) {
		return errors.Validation("execute_at must be in the future")
	}

	return nil
}

func validateIntervalSchedule(config json.RawMessage) error {
	var cfg entities.IntervalSchedule
	if err := json.Unmarshal(config, &cfg); err != nil {
		return errors.Validation("invalid interval schedule config")
	}

	if cfg.Value <= 0 {
		return errors.Validation("interval value must be positive")
	}

	validUnits := map[string]bool{
		"hours":  true,
		"days":   true,
		"weeks":  true,
		"months": true,
	}

	if !validUnits[cfg.Unit] {
		return errors.Validation("interval unit must be one of: hours, days, weeks, months")
	}

	if cfg.Unit == "hours" && cfg.Value > 24 {
		return errors.Validation("interval cannot exceed 24 hours")
	}

	return nil
}

func validateDailySchedule(config json.RawMessage) error {
	var cfg entities.DailySchedule
	if err := json.Unmarshal(config, &cfg); err != nil {
		return errors.Validation("invalid daily schedule config")
	}

	if cfg.Hour < 0 || cfg.Hour > 23 {
		return errors.Validation("hour must be between 0 and 23")
	}

	if cfg.Minute < 0 || cfg.Minute > 59 {
		return errors.Validation("minute must be between 0 and 59")
	}

	if len(cfg.Weekdays) == 0 {
		return errors.Validation("at least one weekday must be specified")
	}

	for _, day := range cfg.Weekdays {
		if day < 1 || day > 7 {
			return errors.Validation("weekdays must be between 1 (Monday) and 7 (Sunday)")
		}
	}

	return nil
}
