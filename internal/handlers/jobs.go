package handlers

import (
	"github.com/davidmovas/postulator/internal/domain/jobs"
	"github.com/davidmovas/postulator/internal/dto"
	"github.com/davidmovas/postulator/pkg/ctx"
)

type JobsHandler struct {
	service jobs.Service
}

func NewJobsHandler(service jobs.Service) *JobsHandler {
	return &JobsHandler{
		service: service,
	}
}

func (h *JobsHandler) CreateJob(job *dto.Job) *dto.Response[string] {
	entity, err := job.ToEntity()
	if err != nil {
		return fail[string](err)
	}

	if err = h.service.CreateJob(ctx.FastCtx(), entity); err != nil {
		return fail[string](err)
	}

	return ok("Job created successfully")
}

func (h *JobsHandler) GetJob(id int64) *dto.Response[*dto.Job] {
	job, err := h.service.GetJob(ctx.FastCtx(), id)
	if err != nil {
		return fail[*dto.Job](err)
	}

	return ok(dto.NewJob(job))
}

func (h *JobsHandler) ListJobs() *dto.Response[[]*dto.Job] {
	listJobs, err := h.service.ListJobs(ctx.FastCtx())
	if err != nil {
		return fail[[]*dto.Job](err)
	}

	var dtoJobs []*dto.Job
	for _, job := range listJobs {
		dtoJobs = append(dtoJobs, dto.NewJob(job))
	}

	return ok(dtoJobs)
}

func (h *JobsHandler) UpdateJob(job *dto.Job) *dto.Response[string] {
	entity, err := job.ToEntity()
	if err != nil {
		return fail[string](err)
	}

	if err = h.service.UpdateJob(ctx.FastCtx(), entity); err != nil {
		return fail[string](err)
	}

	return ok("Job updated successfully")
}

func (h *JobsHandler) DeleteJob(id int64) *dto.Response[string] {
	if err := h.service.DeleteJob(ctx.FastCtx(), id); err != nil {
		return fail[string](err)
	}

	return ok("Job deleted")
}

func (h *JobsHandler) PauseJob(id int64) *dto.Response[string] {
	if err := h.service.PauseJob(ctx.FastCtx(), id); err != nil {
		return fail[string](err)
	}

	return ok("Job paused")
}

func (h *JobsHandler) ResumeJob(id int64) *dto.Response[string] {
	if err := h.service.ResumeJob(ctx.FastCtx(), id); err != nil {
		return fail[string](err)
	}

	return ok("Job resumed")
}

func (h *JobsHandler) ExecuteManually(jobID int64) *dto.Response[string] {
	if err := h.service.ExecuteManually(ctx.FastCtx(), jobID); err != nil {
		return fail[string](err)
	}

	return ok("Job executed")
}
