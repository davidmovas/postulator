package dto

import (
	"Postulator/internal/domain/job"
	"time"
)

type Job struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	SiteID       int64  `json:"siteId"`
	CategoryID   int64  `json:"categoryId"`
	PromptID     int64  `json:"promptId"`
	AIProviderID int64  `json:"aiProviderId"`
	AIModel      string `json:"aiModel"`

	RequiresValidation bool    `json:"requiresValidation"`
	ScheduleType       string  `json:"scheduleType"`
	ScheduleTime       *string `json:"scheduleTime,omitempty"`
	ScheduleDay        *int    `json:"scheduleDay,omitempty"`
	JitterEnabled      bool    `json:"jitterEnabled"`
	JitterMinutes      int     `json:"jitterMinutes"`
	Status             string  `json:"status"`
	LastRunAt          *string `json:"lastRunAt,omitempty"`
	NextRunAt          *string `json:"nextRunAt,omitempty"`
	CreatedAt          string  `json:"createdAt"`
	UpdatedAt          string  `json:"updatedAt"`
}

type Execution struct {
	ID               int64   `json:"id"`
	JobID            int64   `json:"jobId"`
	TopicID          int64   `json:"topicId"`
	GeneratedTitle   *string `json:"generatedTitle,omitempty"`
	GeneratedContent *string `json:"generatedContent,omitempty"`
	Status           string  `json:"status"`
	ErrorMessage     *string `json:"errorMessage,omitempty"`
	ArticleID        *int64  `json:"articleId,omitempty"`
	StartedAt        string  `json:"startedAt"`
	GeneratedAt      *string `json:"generatedAt,omitempty"`
	ValidatedAt      *string `json:"validatedAt,omitempty"`
	PublishedAt      *string `json:"publishedAt,omitempty"`
}

func FromJob(e *job.Job) *Job {
	if e == nil {
		return nil
	}
	toTimeStr := func(t *time.Time) *string {
		if t == nil {
			return nil
		}
		v := t.UTC().Format(timeLayout)
		return &v
	}
	var scheduleTime *string
	if e.ScheduleTime != nil {
		v := e.ScheduleTime.UTC().Format("15:04:05")
		scheduleTime = &v
	}
	return &Job{
		ID:                 e.ID,
		Name:               e.Name,
		SiteID:             e.SiteID,
		CategoryID:         e.CategoryID,
		PromptID:           e.PromptID,
		AIProviderID:       e.AIProviderID,
		AIModel:            e.AIModel,
		RequiresValidation: e.RequiresValidation,
		ScheduleType:       string(e.ScheduleType),
		ScheduleTime:       scheduleTime,
		ScheduleDay:        e.ScheduleDay,
		JitterEnabled:      e.JitterEnabled,
		JitterMinutes:      e.JitterMinutes,
		Status:             string(e.Status),
		LastRunAt:          toTimeStr(e.LastRunAt),
		NextRunAt:          toTimeStr(e.NextRunAt),
		CreatedAt:          e.CreatedAt.UTC().Format(timeLayout),
		UpdatedAt:          e.UpdatedAt.UTC().Format(timeLayout),
	}
}

func FromJobs(items []*job.Job) []*Job {
	out := make([]*Job, 0, len(items))
	for _, it := range items {
		out = append(out, FromJob(it))
	}
	return out
}

func FromExecution(e *job.Execution) *Execution {
	if e == nil {
		return nil
	}
	toTimeStr := func(t *time.Time) *string {
		if t == nil {
			return nil
		}
		v := t.UTC().Format(timeLayout)
		return &v
	}
	return &Execution{
		ID:               e.ID,
		JobID:            e.JobID,
		TopicID:          e.TopicID,
		GeneratedTitle:   e.GeneratedTitle,
		GeneratedContent: e.GeneratedContent,
		Status:           string(e.Status),
		ErrorMessage:     e.ErrorMessage,
		ArticleID:        e.ArticleID,
		StartedAt:        e.StartedAt.UTC().Format(timeLayout),
		GeneratedAt:      toTimeStr(e.GeneratedAt),
		ValidatedAt:      toTimeStr(e.ValidatedAt),
		PublishedAt:      toTimeStr(e.PublishedAt),
	}
}

func FromExecutions(items []*job.Execution) []*Execution {
	out := make([]*Execution, 0, len(items))
	for _, it := range items {
		out = append(out, FromExecution(it))
	}
	return out
}
