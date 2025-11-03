package dto

import (
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

type Site struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	URL             string `json:"url"`
	WPUsername      string `json:"wpUsername"`
	WPPassword      string `json:"wpPassword"`
	Status          string `json:"status"`
	LastHealthCheck string `json:"lastHealthCheck"`
	HealthStatus    string `json:"healthStatus"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

func NewSite(site *entities.Site) *Site {
	s := &Site{}
	return s.FromEntity(site)
}

func (d *Site) ToEntity() (*entities.Site, error) {
	createdAt, err := StringToTime(d.CreatedAt)
	if err != nil {
		return nil, err
	}

	updatedAt, err := StringToTime(d.UpdatedAt)
	if err != nil {
		return nil, err
	}

	var lastHealthCheck *time.Time
	if d.LastHealthCheck != "" {
		var lastHealthCheckTime time.Time
		lastHealthCheckTime, err = StringToTime(d.LastHealthCheck)
		if err != nil {
			return nil, err
		}
		lastHealthCheck = &lastHealthCheckTime
	}

	return &entities.Site{
		ID:              d.ID,
		Name:            d.Name,
		URL:             d.URL,
		WPUsername:      d.WPUsername,
		WPPassword:      d.WPPassword,
		Status:          entities.Status(d.Status),
		LastHealthCheck: lastHealthCheck,
		HealthStatus:    entities.HealthStatus(d.HealthStatus),
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}, nil
}

func (d *Site) FromEntity(entity *entities.Site) *Site {
	d.ID = entity.ID
	d.Name = entity.Name
	d.URL = entity.URL
	d.WPUsername = entity.WPUsername
	d.WPPassword = entity.WPPassword
	d.Status = string(entity.Status)
	d.HealthStatus = string(entity.HealthStatus)
	d.CreatedAt = TimeToString(entity.CreatedAt)
	d.UpdatedAt = TimeToString(entity.UpdatedAt)

	if entity.LastHealthCheck != nil {
		d.LastHealthCheck = TimeToString(*entity.LastHealthCheck)
	} else {
		d.LastHealthCheck = ""
	}

	return d
}
