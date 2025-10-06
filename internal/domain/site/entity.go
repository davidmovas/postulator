package site

import "time"

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusError    Status = "error"
)

type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
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

type Category struct {
	ID           int64
	SiteID       int64
	WPCategoryID int
	Name         string
	Slug         string
	CreatedAt    time.Time
}
