package runs

import (
	"encoding/json"
	"time"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type Run struct {
	ID              string              `json:"id"`
	SiteID          string              `json:"siteId"`
	Kind            string              `json:"kind"`
	Status          string              `json:"status"`
	Targets         []string            `json:"targets"`
	Recipe          []template.StepSpec `json:"recipe"`
	TemplateID      string              `json:"templateId"`
	TemplateVersion int                 `json:"templateVersion"`
	PublishMode     string              `json:"publishMode"`
	Budget          run.Budget          `json:"budget"`
	Stats           run.Stats           `json:"stats"`
	CreatedBy       string              `json:"createdBy"`
	ParentRunID     *string             `json:"parentRunId"`
	PauseReason     string              `json:"pauseReason"`
	Error           string              `json:"error"`
	DeadlineAt      dto.Time            `json:"deadlineAt"`
	CreatedAt       dto.Time            `json:"createdAt"`
	StartedAt       dto.Time            `json:"startedAt"`
	FinishedAt      dto.Time            `json:"finishedAt"`
}

type Item struct {
	ID                 string         `json:"id"`
	RunID              string         `json:"runId"`
	SiteID             string         `json:"siteId"`
	TargetID           string         `json:"targetId"`
	Status             string         `json:"status"`
	CurrentStep        string         `json:"currentStep"`
	Seq                int            `json:"seq"`
	BlockedBy          string         `json:"blockedBy"`
	Attempts           int            `json:"attempts"`
	PauseReason        string         `json:"pauseReason"`
	Error              string         `json:"error"`
	Note               string         `json:"note"`
	WaitingFor         *AwaitedParent `json:"waitingFor"`
	Retryable          bool           `json:"retryable"`
	RetryBlockedReason string         `json:"retryBlockedReason"`
	WakeAt             dto.Time       `json:"wakeAt"`
	CreatedAt          dto.Time       `json:"createdAt"`
	UpdatedAt          dto.Time       `json:"updatedAt"`
	FinishedAt         dto.Time       `json:"finishedAt"`
}

type AwaitedParent struct {
	PageID     string `json:"pageId"`
	Path       string `json:"path"`
	ItemID     string `json:"itemId"`
	ItemStatus string `json:"itemStatus"`
	Step       string `json:"step"`
}

type Event struct {
	RunID   string          `json:"runId"`
	Seq     int64           `json:"seq"`
	Type    string          `json:"type"`
	At      dto.Time        `json:"at"`
	Payload json.RawMessage `json:"payload"`
}

type Artifact struct {
	ID        string   `json:"id"`
	RunID     string   `json:"runId"`
	ItemID    string   `json:"itemId"`
	Step      string   `json:"step"`
	Kind      string   `json:"kind"`
	Content   string   `json:"content"`
	Size      int      `json:"size"`
	Hash      string   `json:"hash"`
	Purged    bool     `json:"purged"`
	ExpiresAt dto.Time `json:"expiresAt"`
	CreatedAt dto.Time `json:"createdAt"`
}

type ArtifactSummary struct {
	ID        string   `json:"id"`
	RunID     string   `json:"runId"`
	ItemID    string   `json:"itemId"`
	Step      string   `json:"step"`
	Kind      string   `json:"kind"`
	Size      int      `json:"size"`
	Hash      string   `json:"hash"`
	Purged    bool     `json:"purged"`
	ExpiresAt dto.Time `json:"expiresAt"`
	CreatedAt dto.Time `json:"createdAt"`
}

func timeOf(at *time.Time) dto.Time {
	if at == nil {
		return dto.Time{}
	}
	return dto.NewTime(*at)
}

func runView(record run.Run) Run {
	return Run{
		ID:              record.ID,
		SiteID:          record.SiteID,
		Kind:            string(record.Kind),
		Status:          string(record.Status),
		Targets:         record.Targets,
		Recipe:          record.Recipe,
		TemplateID:      record.TemplateID,
		TemplateVersion: record.TemplateVersion,
		PublishMode:     string(record.PublishMode),
		Budget:          record.Budget,
		Stats:           record.Stats,
		CreatedBy:       string(record.CreatedBy),
		ParentRunID:     record.ParentRunID,
		PauseReason:     string(record.PauseReason),
		Error:           record.Error,
		DeadlineAt:      dto.NewTime(record.DeadlineAt),
		CreatedAt:       dto.NewTime(record.CreatedAt),
		StartedAt:       timeOf(record.StartedAt),
		FinishedAt:      timeOf(record.FinishedAt),
	}
}

func itemView(item run.Item, blocked run.RetryBlockedReason, awaited *AwaitedParent) Item {
	return Item{
		ID:                 item.ID,
		RunID:              item.RunID,
		SiteID:             item.SiteID,
		TargetID:           item.TargetID,
		Status:             string(item.Status),
		CurrentStep:        item.CurrentStep,
		Seq:                item.Seq,
		BlockedBy:          item.BlockedBy,
		Attempts:           item.Attempts,
		PauseReason:        string(item.PauseReason),
		Error:              item.Error,
		Note:               item.Note,
		WaitingFor:         awaited,
		Retryable:          blocked == "",
		RetryBlockedReason: string(blocked),
		WakeAt:             timeOf(item.WakeAt),
		CreatedAt:          dto.NewTime(item.CreatedAt),
		UpdatedAt:          dto.NewTime(item.UpdatedAt),
		FinishedAt:         timeOf(item.FinishedAt),
	}
}

func eventView(event run.Event) Event {
	return Event{
		RunID:   event.RunID,
		Seq:     event.Seq,
		Type:    event.Type,
		At:      dto.NewTime(event.At),
		Payload: event.Payload,
	}
}

func artifactSummaryView(artifact run.Artifact) ArtifactSummary {
	return ArtifactSummary{
		ID:        artifact.ID,
		RunID:     artifact.RunID,
		ItemID:    artifact.ItemID,
		Step:      artifact.Step,
		Kind:      string(artifact.Kind),
		Size:      artifact.Size,
		Hash:      artifact.Hash,
		Purged:    artifact.Purged,
		ExpiresAt: timeOf(artifact.ExpiresAt),
		CreatedAt: dto.NewTime(artifact.CreatedAt),
	}
}

func artifactView(artifact run.Artifact) Artifact {
	return Artifact{
		ID:        artifact.ID,
		RunID:     artifact.RunID,
		ItemID:    artifact.ItemID,
		Step:      artifact.Step,
		Kind:      string(artifact.Kind),
		Content:   string(artifact.Blob),
		Size:      artifact.Size,
		Hash:      artifact.Hash,
		Purged:    artifact.Purged,
		ExpiresAt: timeOf(artifact.ExpiresAt),
		CreatedAt: dto.NewTime(artifact.CreatedAt),
	}
}
