package handlers

import (
	"Postulator/internal/repository"
	"Postulator/internal/services/gpt"
	"Postulator/internal/services/pipeline"
	"Postulator/internal/services/wordpress"
	"context"
	"time"
)

// Handler contains all Wails API handlers
type Handler struct {
	ctx        context.Context
	gptService *gpt.Service
	wpService  *wordpress.Service
	pipeline   *pipeline.Service
	repo       *repository.Repository
}

// NewHandler creates a new Handler instance
func NewHandler(appContext context.Context, gptService *gpt.Service, wpService *wordpress.Service, pipeline *pipeline.Service, repo *repository.Repository) *Handler {
	return &Handler{
		ctx:        appContext,
		gptService: gptService,
		wpService:  wpService,
		pipeline:   pipeline,
		repo:       repo,
	}
}

func (h *Handler) fastCtx() context.Context {
	taskCtx, cancel := context.WithTimeout(h.ctx, 10*time.Second)
	time.AfterFunc(10*time.Second, cancel)
	return taskCtx
}
