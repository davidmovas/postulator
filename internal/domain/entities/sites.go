package entities

import "time"

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusError    Status = "error"
)

type HealthStatus string

const (
	HealthHealthy   HealthStatus = "healthy"
	HealthUnhealthy HealthStatus = "unhealthy"
	HealthUnknown   HealthStatus = "unknown"
	HealthError     HealthStatus = "error"
)

type Site struct {
	ID              int64
	Name            string
	URL             string
	WPUsername      string
	WPPassword      string
	Status          Status
	LastHealthCheck *time.Time
	AutoHealthCheck bool
	HealthStatus    HealthStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type HealthCheck struct {
	SiteID       int64
	Status       HealthStatus
	Code         int
	StatusCode   string
	ResponseTime time.Duration
	Error        string
}
