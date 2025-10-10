package app

import (
	"Postulator/internal/domain/job"
	"Postulator/internal/dto"
	"Postulator/pkg/ctx"
	"Postulator/pkg/errors"
	"time"
)

func (a *App) CreateJob(j *dto.Job) *dto.Response[string] {
	if j == nil {
		return dtoErr[string](errors.Validation("job payload is required"))
	}

	entity, err := toJobEntity(j)
	if err != nil {
		return dtoErr[string](errors.Validation(err.Error()))
	}

	if err = a.jobSvc.CreateJob(ctx.FastCtx(), entity); err != nil {
		return dtoErr[string](asAppErr(err))
	}

	return &dto.Response[string]{Success: true, Data: "created"}
}

func (a *App) GetJob(id int64) *dto.Response[*dto.Job] {
	res, err := a.jobSvc.GetJob(ctx.FastCtx(), id)
	if err != nil {
		return dtoErr[*dto.Job](asAppErr(err))
	}

	return &dto.Response[*dto.Job]{Success: true, Data: dto.FromJob(res)}
}

func (a *App) ListJobs() *dto.Response[[]*dto.Job] {
	items, err := a.jobSvc.ListJobs(ctx.FastCtx())
	if err != nil {
		return dtoErr[[]*dto.Job](asAppErr(err))
	}

	return &dto.Response[[]*dto.Job]{Success: true, Data: dto.FromJobs(items)}
}

func (a *App) UpdateJob(j *dto.Job) *dto.Response[string] {
	if j == nil {
		return dtoErr[string](errors.Validation("job payload is required"))
	}

	entity, err := toJobEntity(j)
	if err != nil {
		return dtoErr[string](errors.Validation(err.Error()))
	}

	entity.ID = j.ID
	if err = a.jobSvc.UpdateJob(ctx.FastCtx(), entity); err != nil {
		return dtoErr[string](asAppErr(err))
	}

	return &dto.Response[string]{Success: true, Data: "updated"}
}

func (a *App) DeleteJob(id int64) *dto.Response[string] {
	if err := a.jobSvc.DeleteJob(ctx.FastCtx(), id); err != nil {
		return dtoErr[string](asAppErr(err))
	}

	return &dto.Response[string]{Success: true, Data: "deleted"}
}

func (a *App) PauseJob(id int64) *dto.Response[string] {
	if err := a.jobSvc.PauseJob(ctx.FastCtx(), id); err != nil {
		return dtoErr[string](asAppErr(err))
	}

	return &dto.Response[string]{Success: true, Data: "paused"}
}

func (a *App) ResumeJob(id int64) *dto.Response[string] {
	if err := a.jobSvc.ResumeJob(ctx.FastCtx(), id); err != nil {
		return dtoErr[string](asAppErr(err))
	}

	return &dto.Response[string]{Success: true, Data: "resumed"}
}

func (a *App) ExecuteJobManually(id int64) *dto.Response[string] {
	if err := a.jobSvc.ExecuteJobManually(ctx.FastCtx(), id); err != nil {
		return dtoErr[string](asAppErr(err))
	}

	return &dto.Response[string]{Success: true, Data: "executed"}
}

func (a *App) GetPendingValidations() *dto.Response[[]*dto.Execution] {
	items, err := a.jobSvc.GetPendingValidations(ctx.FastCtx())
	if err != nil {
		return dtoErr[[]*dto.Execution](asAppErr(err))
	}

	return &dto.Response[[]*dto.Execution]{Success: true, Data: dto.FromExecutions(items)}
}

func (a *App) ValidateExecution(execID int64, approved bool) *dto.Response[string] {
	if err := a.jobSvc.ValidateExecution(ctx.FastCtx(), execID, approved); err != nil {
		return dtoErr[string](asAppErr(err))
	}

	return &dto.Response[string]{Success: true, Data: "validated"}
}

func toJobEntity(d *dto.Job) (*job.Job, error) {
	var scheduleTime *time.Time
	if d.ScheduleTime != nil && *d.ScheduleTime != "" {
		parsed, err := time.Parse("15:04:05", *d.ScheduleTime)
		if err != nil {
			return nil, err
		}
		now := time.Now()
		st := time.Date(now.Year(), now.Month(), now.Day(), parsed.Hour(), parsed.Minute(), parsed.Second(), 0, now.Location())
		scheduleTime = &st
	}

	return &job.Job{
		ID:                 d.ID,
		Name:               d.Name,
		SiteID:             d.SiteID,
		CategoryID:         d.CategoryID,
		PromptID:           d.PromptID,
		AIProviderID:       d.AIProviderID,
		AIModel:            d.AIModel,
		RequiresValidation: d.RequiresValidation,
		ScheduleType:       job.ScheduleType(d.ScheduleType),
		ScheduleTime:       scheduleTime,
		ScheduleDay:        d.ScheduleDay,
		JitterEnabled:      d.JitterEnabled,
		JitterMinutes:      d.JitterMinutes,
		Status:             job.Status(d.Status),
	}, nil
}
