package entities

import (
	"time"

	"github.com/davidmovas/postulator/pkg/errors"
)

const (
	SettingsKeyHealthCheck = "health_check"
)

type HealthCheckSettings struct {
	Enabled            bool `json:"enabled"`
	IntervalMinutes    int  `json:"interval_minutes"`
	MinIntervalMinutes int  `json:"min_interval_minutes"`
	NotifyWhenHidden   bool `json:"notify_when_hidden"`
	NotifyAlways       bool `json:"notify_always"`
	NotifyWithSound    bool `json:"notify_with_sound"`
	NotifyOnRecover    bool `json:"notify_on_recover"`
}

func DefaultHealthCheckSettings() *HealthCheckSettings {
	return &HealthCheckSettings{
		Enabled:            false,
		IntervalMinutes:    5,
		MinIntervalMinutes: 1,
		NotifyWhenHidden:   true,
		NotifyAlways:       false,
		NotifyWithSound:    true,
		NotifyOnRecover:    true,
	}
}

func (s *HealthCheckSettings) Validate() error {
	if s.IntervalMinutes < s.MinIntervalMinutes {
		return errors.Validation("Interval cannot be less than minimum interval")
	}
	return nil
}

type HealthCheckHistory struct {
	ID             int64
	SiteID         int64
	CheckedAt      time.Time
	Status         HealthStatus
	ResponseTimeMs int
	StatusCode     int
	ErrorMessage   string
}

type AppError struct {
	Code    string
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}
