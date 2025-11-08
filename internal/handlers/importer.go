package handlers

import (
	"github.com/davidmovas/postulator/internal/dto"
	"github.com/davidmovas/postulator/internal/infra/importer"
	"github.com/davidmovas/postulator/pkg/ctx"
)

type ImporterHandler struct {
	service importer.Service
}

func NewImporterHandler(service importer.Service) *ImporterHandler {
	return &ImporterHandler{service: service}
}

func (h *ImporterHandler) ImportTopics(filePath string) *dto.Response[*dto.ImportResult] {
	res, err := h.service.ImportTopics(ctx.FastCtx(), filePath)
	if err != nil {
		return fail[*dto.ImportResult](err)
	}

	return ok(dto.NewImportResult(res))
}

func (h *ImporterHandler) ImportAndAssignToSite(filePath string, siteID int64) *dto.Response[*dto.ImportResult] {
	res, err := h.service.ImportAndAssignToSite(ctx.FastCtx(), filePath, siteID)
	if err != nil {
		return fail[*dto.ImportResult](err)
	}

	return ok(dto.NewImportResult(res))
}
