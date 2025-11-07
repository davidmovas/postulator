package dto

import "github.com/davidmovas/postulator/internal/domain/entities"

type HealthCheckSettings struct {
	Enabled            bool `json:"enabled"`
	IntervalMinutes    int  `json:"interval_minutes"`
	MinIntervalMinutes int  `json:"min_interval_minutes"`
	NotifyWhenHidden   bool `json:"notify_when_hidden"`
	NotifyAlways       bool `json:"notify_always"`
	NotifyWithSound    bool `json:"notify_with_sound"`
	NotifyOnRecover    bool `json:"notify_on_recover"`
}

func NewHealthCheckSettings(s *entities.HealthCheckSettings) *HealthCheckSettings {
	h := &HealthCheckSettings{}
	return h.FromEntity(s)
}

func (s *HealthCheckSettings) ToEntity() (*entities.HealthCheckSettings, error) {
	return &entities.HealthCheckSettings{
		Enabled:            s.Enabled,
		IntervalMinutes:    s.IntervalMinutes,
		MinIntervalMinutes: s.MinIntervalMinutes,
		NotifyWhenHidden:   s.NotifyWhenHidden,
		NotifyAlways:       s.NotifyAlways,
		NotifyWithSound:    s.NotifyWithSound,
		NotifyOnRecover:    s.NotifyOnRecover,
	}, nil
}

func (s *HealthCheckSettings) FromEntity(e *entities.HealthCheckSettings) *HealthCheckSettings {
	s.Enabled = e.Enabled
	s.IntervalMinutes = e.IntervalMinutes
	s.MinIntervalMinutes = e.MinIntervalMinutes
	s.NotifyWhenHidden = e.NotifyWhenHidden
	s.NotifyAlways = e.NotifyAlways
	s.NotifyWithSound = e.NotifyWithSound
	s.NotifyOnRecover = e.NotifyOnRecover
	return s
}
