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
)

type Site struct {
	ID              int64
	Name            string
	URL             string
	WPUsername      string
	WPPassword      string
	Status          Status
	LastHealthCheck *time.Time
	HealthStatus    HealthStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
