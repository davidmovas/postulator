package reports

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func (s *Service) PageReport(ctx context.Context, req PageReportRequest) (PageReportResponse, error) {
	pageID := strings.TrimSpace(req.PageID)
	if pageID == "" {
		return PageReportResponse{}, invalid("a page report needs a page", "pageId")
	}

	page, err := s.pages.Get(ctx, pageID)
	if err != nil {
		return PageReportResponse{}, err
	}

	items, err := s.items.ByTarget(ctx, pageID, recentItems)
	if err != nil {
		return PageReportResponse{}, err
	}
	if len(items) == 0 {
		return PageReportResponse{}, errors.New(errors.NotFound, "no run has touched this page yet").
			WithDetail("pageId", pageID)
	}

	newest := items[0]
	artifacts, err := s.artifacts.ByItem(ctx, newest.ID)
	if err != nil {
		return PageReportResponse{}, err
	}

	report := PageReportResponse{
		PageID: page.ID, Path: page.Path, RunID: newest.RunID, ItemID: newest.ID,
		Status: string(newest.Status),
	}
	if newest.FinishedAt != nil {
		finished := dto.Time(*newest.FinishedAt)
		report.FinishedAt = &finished
	}
	report.Validation = latest(artifacts, run.ArtifactValidationReport)
	report.Judge = latest(artifacts, run.ArtifactJudgeReport)
	report.Publish = latest(artifacts, run.ArtifactPublishResult)
	report.Relink = latest(artifacts, run.ArtifactRelinkResult)
	return report, nil
}

func (s *Service) RunReport(ctx context.Context, req RunReportRequest) (RunReportResponse, error) {
	runID := strings.TrimSpace(req.RunID)
	if runID == "" {
		return RunReportResponse{}, invalid("a run report needs a run", "runId")
	}

	record, err := s.runs.Get(ctx, runID)
	if err != nil {
		return RunReportResponse{}, err
	}
	items, err := s.items.ByRun(ctx, runID)
	if err != nil {
		return RunReportResponse{}, err
	}

	report := RunReportResponse{
		RunID: record.ID, SiteID: record.SiteID, Kind: string(record.Kind), Status: string(record.Status),
		Stats: RunStats{
			Items: record.Stats.Items, Done: record.Stats.Done, Failed: record.Stats.Failed,
			Tokens: record.Stats.Tokens, USD: record.Stats.USD,
		},
		Items: make([]ItemReport, 0, len(items)),
	}

	for i := range items {
		artifacts, artifactErr := s.artifacts.ByItem(ctx, items[i].ID)
		if artifactErr != nil {
			return RunReportResponse{}, artifactErr
		}
		report.Items = append(report.Items, ItemReport{
			ItemID: items[i].ID, PageID: items[i].TargetID, Status: string(items[i].Status),
			Error: items[i].Error, Report: latest(artifacts, run.ArtifactFinalReport),
		})
	}
	return report, nil
}

func latest(artifacts []run.Artifact, kind run.ArtifactKind) json.RawMessage {
	var newest *run.Artifact
	for i := range artifacts {
		if artifacts[i].Kind != kind || artifacts[i].Purged || len(artifacts[i].Blob) == 0 {
			continue
		}
		if newest == nil || !artifacts[i].CreatedAt.Before(newest.CreatedAt) {
			newest = &artifacts[i]
		}
	}
	if newest == nil {
		return nil
	}
	return json.RawMessage(newest.Blob)
}
