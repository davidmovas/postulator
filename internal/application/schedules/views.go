package schedules

import (
	"time"

	"github.com/davidmovas/postulator/internal/domain/schedule"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type Schedule struct {
	ID              string   `json:"id"`
	SiteID          string   `json:"siteId"`
	Name            string   `json:"name"`
	Cron            string   `json:"cron,omitempty"`
	IntervalMinutes int      `json:"intervalMinutes,omitempty"`
	EntityID        *string  `json:"entityId,omitempty"`
	Status          string   `json:"status,omitempty"`
	Limit           int      `json:"limit"`
	TemplateID      string   `json:"templateId,omitempty"`
	Steps           []string `json:"steps"`
	PublishMode     string   `json:"publishMode"`
	MaxUSD          float64  `json:"maxUsd"`
	MaxTokens       int      `json:"maxTokens"`
	Enabled         bool     `json:"enabled"`
	NextRunAt       dto.Time `json:"nextRunAt"`
	LastRunID       *string  `json:"lastRunId,omitempty"`
	CreatedBy       string   `json:"createdBy"`
	CreatedAt       dto.Time `json:"createdAt"`
	UpdatedAt       dto.Time `json:"updatedAt"`
}

func view(s schedule.Schedule) Schedule {
	minutes := 0
	if s.Interval != nil {
		minutes = int(s.Interval.Minutes())
	}

	steps := make([]string, 0, len(s.Recipe))
	for i := range s.Recipe {
		steps = append(steps, s.Recipe[i].Name)
	}

	return Schedule{
		ID: s.ID, SiteID: s.SiteID, Name: s.Name, Cron: s.Cron, IntervalMinutes: minutes,
		EntityID: s.Query.EntityID, Status: s.Query.Status, Limit: s.Query.Limit, TemplateID: s.TemplateID,
		Steps: steps, PublishMode: string(s.PublishMode), MaxUSD: s.Budget.MaxUSD,
		MaxTokens: s.Budget.MaxTokens, Enabled: s.Enabled, NextRunAt: timeOf(s.NextRunAt),
		LastRunID: s.LastRunID, CreatedBy: string(s.CreatedBy), CreatedAt: dto.NewTime(s.CreatedAt),
		UpdatedAt: dto.NewTime(s.UpdatedAt),
	}
}

func recipeOf(steps []string) []template.StepSpec {
	recipe := make([]template.StepSpec, 0, len(steps))
	for _, step := range steps {
		recipe = append(recipe, template.StepSpec{Name: step, Enabled: true})
	}
	return recipe
}

func timeOf(at *time.Time) dto.Time {
	if at == nil {
		return dto.Time{}
	}
	return dto.NewTime(*at)
}
