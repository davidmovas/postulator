package handlers

import (
	"Postulator/internal/dto"
	"fmt"
)

// Pipeline and Content Generation Handlers

// GenerateAndPublish generates and publishes an article based on the request
func (h *Handler) GenerateAndPublish(req dto.GeneratePublishRequest) (*dto.ArticleResponse, error) {
	ctx := h.fastCtx()

	if req.SiteID <= 0 {
		return nil, fmt.Errorf("invalid site ID")
	}

	// Call pipeline service to generate and publish article
	result, err := h.pipeline.GenerateAndPublish(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate and publish article: %w", err)
	}

	return result, nil
}

// Job Management Handlers

// CreatePublishJob creates a new posting job for background processing
func (h *Handler) CreatePublishJob(req dto.GeneratePublishRequest) (*dto.JobResponse, error) {
	ctx := h.fastCtx()

	if req.SiteID <= 0 {
		return nil, fmt.Errorf("invalid site ID")
	}

	// Call pipeline service to create job
	result, err := h.pipeline.CreatePublishJob(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create publish job: %w", err)
	}

	return result, nil
}

// GetJobs retrieves a paginated list of jobs
func (h *Handler) GetJobs(req dto.PaginationRequest) (*dto.JobListResponse, error) {
	ctx := h.fastCtx()

	// Set defaults if needed
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}

	// Call pipeline service to get jobs
	result, err := h.pipeline.GetJobs(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs: %w", err)
	}

	return result, nil
}

// GetJob retrieves a specific job by ID
func (h *Handler) GetJob(jobID int64) (*dto.JobResponse, error) {
	ctx := h.fastCtx()

	if jobID <= 0 {
		return nil, fmt.Errorf("invalid job ID")
	}

	// Call pipeline service to get job
	result, err := h.pipeline.GetJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return result, nil
}
