package dto

import (
	"encoding/json"
	"time"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

type Job struct {
	ID                 int64             `json:"id"`
	Name               string            `json:"name"`
	SiteID             int64             `json:"siteId"`
	PromptID           int64             `json:"promptId"`
	AIProviderID       int64             `json:"aiProviderId"`
	PlaceholdersValues map[string]string `json:"placeholdersValues"`
	TopicStrategy      string            `json:"topicStrategy"`
	CategoryStrategy   string            `json:"categoryStrategy"`
	RequiresValidation bool              `json:"requiresValidation"`
	JitterEnabled      bool              `json:"jitterEnabled"`
	JitterMinutes      int               `json:"jitterMinutes"`
	Status             string            `json:"status"`
	CreatedAt          string            `json:"createdAt"`
	UpdatedAt          string            `json:"updatedAt"`
	Schedule           *Schedule         `json:"schedule"`
	State              *State            `json:"state"`
	Categories         []int64           `json:"categories"`
	Topics             []int64           `json:"topics"`
}

func NewJob(entity *entities.Job) *Job {
	j := &Job{}
	return j.FromEntity(entity)
}

func (d *Job) ToEntity() (*entities.Job, error) {
	createdAt, err := StringToTime(d.CreatedAt)
	if err != nil {
		return nil, err
	}

	updatedAt, err := StringToTime(d.UpdatedAt)
	if err != nil {
		return nil, err
	}

	var schedule *entities.Schedule
	if d.Schedule != nil {
		schedule, err = d.Schedule.ToEntity()
		if err != nil {
			return nil, err
		}
	}

	var state *entities.State
	if d.State != nil {
		state = d.State.ToEntity()
	}

	return &entities.Job{
		ID:                 d.ID,
		Name:               d.Name,
		SiteID:             d.SiteID,
		PromptID:           d.PromptID,
		AIProviderID:       d.AIProviderID,
		PlaceholdersValues: d.PlaceholdersValues,
		TopicStrategy:      entities.TopicStrategy(d.TopicStrategy),
		CategoryStrategy:   entities.CategoryStrategy(d.CategoryStrategy),
		RequiresValidation: d.RequiresValidation,
		JitterEnabled:      d.JitterEnabled,
		JitterMinutes:      d.JitterMinutes,
		Status:             entities.JobStatus(d.Status),
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
		Schedule:           schedule,
		State:              state,
		Categories:         d.Categories,
		Topics:             d.Topics,
	}, nil
}

func (d *Job) FromEntity(entity *entities.Job) *Job {
	d.ID = entity.ID
	d.Name = entity.Name
	d.SiteID = entity.SiteID
	d.PromptID = entity.PromptID
	d.AIProviderID = entity.AIProviderID
	d.PlaceholdersValues = entity.PlaceholdersValues
	d.TopicStrategy = string(entity.TopicStrategy)
	d.CategoryStrategy = string(entity.CategoryStrategy)
	d.RequiresValidation = entity.RequiresValidation
	d.JitterEnabled = entity.JitterEnabled
	d.JitterMinutes = entity.JitterMinutes
	d.Status = string(entity.Status)
	d.CreatedAt = TimeToString(entity.CreatedAt)
	d.UpdatedAt = TimeToString(entity.UpdatedAt)
	d.Schedule = NewSchedule(entity.Schedule)
	d.State = NewState(entity.State)
	d.Categories = entity.Categories
	d.Topics = entity.Topics
	return d
}

type Schedule struct {
	Type   string `json:"type"`
	Config any    `json:"config"`
}

type OnceSchedule struct {
	ExecuteAt string `json:"executeAt"`
}

type IntervalSchedule struct {
	Value   int     `json:"value"`
	Unit    string  `json:"unit"`
	StartAt *string `json:"startAt,omitempty"`
}

type DailySchedule struct {
	Hour     int   `json:"hour"`
	Minute   int   `json:"minute"`
	Weekdays []int `json:"weekdays"`
}

func NewSchedule(entity *entities.Schedule) *Schedule {
	if entity == nil {
		return nil
	}
	s := &Schedule{}
	return s.FromEntity(entity)
}

func (d *Schedule) ToEntity() (*entities.Schedule, error) {
	var configJSON []byte
	var err error

	if d.Config != nil {
		configJSON, err = json.Marshal(d.Config)
		if err != nil {
			return nil, err
		}
	}

	return &entities.Schedule{
		Type:   entities.ScheduleType(d.Type),
		Config: configJSON,
	}, nil
}

func (d *Schedule) FromEntity(entity *entities.Schedule) *Schedule {
	if entity == nil {
		return nil
	}

	d.Type = string(entity.Type)

	if len(entity.Config) > 0 {
		switch entities.ScheduleType(d.Type) {
		case entities.ScheduleOnce:
			var config OnceSchedule
			if err := json.Unmarshal(entity.Config, &config); err == nil {
				d.Config = config
			}
		case entities.ScheduleInterval:
			var config IntervalSchedule
			if err := json.Unmarshal(entity.Config, &config); err == nil {
				d.Config = config
			}
		case entities.ScheduleDaily:
			var config DailySchedule
			if err := json.Unmarshal(entity.Config, &config); err == nil {
				d.Config = config
			}
		case entities.ScheduleManual:
			d.Config = nil
		}
	}

	return d
}

type State struct {
	JobID             int64   `json:"jobId"`
	LastRunAt         *string `json:"lastRunAt"`
	NextRunAt         *string `json:"nextRunAt"`
	TotalExecutions   int     `json:"totalExecutions"`
	FailedExecutions  int     `json:"failedExecutions"`
	LastCategoryIndex int     `json:"lastCategoryIndex"`
}

func NewState(entity *entities.State) *State {
	if entity == nil {
		return nil
	}
	s := &State{}
	return s.FromEntity(entity)
}

func (d *State) ToEntity() *entities.State {
	var lastRunAt, nextRunAt *time.Time

	if d.LastRunAt != nil {
		lastRunAtTime, _ := StringToTime(*d.LastRunAt)
		lastRunAt = &lastRunAtTime
	}

	if d.NextRunAt != nil {
		nextRunAtTime, _ := StringToTime(*d.NextRunAt)
		nextRunAt = &nextRunAtTime
	}

	return &entities.State{
		JobID:             d.JobID,
		LastRunAt:         lastRunAt,
		NextRunAt:         nextRunAt,
		TotalExecutions:   d.TotalExecutions,
		FailedExecutions:  d.FailedExecutions,
		LastCategoryIndex: d.LastCategoryIndex,
	}
}

func (d *State) FromEntity(entity *entities.State) *State {
	d.JobID = entity.JobID
	d.TotalExecutions = entity.TotalExecutions
	d.FailedExecutions = entity.FailedExecutions
	d.LastCategoryIndex = entity.LastCategoryIndex

	if entity.LastRunAt != nil {
		lastRunAt := TimeToString(*entity.LastRunAt)
		d.LastRunAt = &lastRunAt
	} else {
		d.LastRunAt = nil
	}

	if entity.NextRunAt != nil {
		nextRunAt := TimeToString(*entity.NextRunAt)
		d.NextRunAt = &nextRunAt
	} else {
		d.NextRunAt = nil
	}

	return d
}
