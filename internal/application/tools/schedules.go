package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/schedules"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	schedulesName = "schedules"
	opEnable      = "enable"
	opDisable     = "disable"
	opRunNow      = "run_now"
)

type schedulesArgs struct {
	Op              string   `json:"op" enum:"list,get,create,update,delete,enable,disable,run_now" description:"A schedule starts a run on a cron expression or an interval; run_now starts it at once"`
	ScheduleID      string   `json:"scheduleId,omitempty" description:"get, update, delete, enable, disable, run_now: the schedule"`
	Name            *string  `json:"name,omitempty" description:"create, update: the name, 2-4 words"`
	Cron            *string  `json:"cron,omitempty" description:"create, update: five-field cron in UTC, or a descriptor like @daily"`
	IntervalMinutes *int     `json:"intervalMinutes,omitempty" minimum:"1" description:"create, update: minutes between runs, instead of cron"`
	EntityID        *string  `json:"entityId,omitempty" description:"create, update: only pages of this entity"`
	Status          *string  `json:"status,omitempty" enum:"planned,exists,published,archived" description:"create, update: only pages in this state"`
	MaxPages        *int     `json:"maxPages,omitempty" minimum:"1" description:"create, update: the most pages a firing takes"`
	TemplateID      *string  `json:"templateId,omitempty" description:"create, update: the template to move pages to; default their own"`
	Steps           []string `json:"steps,omitempty" enum:"resolve_context,generate_body,generate_meta,generate_fields,insert_links,repair_links,generate_images,validate,judge,publish,relink_neighbors,sync_back,report" description:"create, update: the steps in order; default the template's recipe"`
	PublishMode     *string  `json:"publishMode,omitempty" enum:"draft,publish" description:"create, update: default draft"`
	MaxUSD          *float64 `json:"maxUsd,omitempty" minimum:"0" description:"create, update: dollar ceiling per run"`
	MaxTokens       *int     `json:"maxTokens,omitempty" minimum:"0" description:"create, update: token ceiling per run"`
	Enabled         *bool    `json:"enabled,omitempty" description:"create: arm it now; list: only armed, or only disarmed, schedules"`
	Cursor          string   `json:"cursor,omitempty" description:"list: nextCursor of the last call"`
	Limit           int      `json:"limit,omitempty" description:"list: default 50"`
}

func schedulesTool(deps Deps) Tool {
	tool := newSiteTool(Def{
		Name:        schedulesName,
		Description: "Read and change the site's schedules, each starting a run over its pages on a cron or an interval.",
		Risk:        RiskDangerous,
	}, func(ctx context.Context, b Binding, in schedulesArgs) (any, error) {
		return deps.manageSchedules(ctx, b, &in)
	})
	tool = describing(tool, func(ctx context.Context, _ Binding, in schedulesArgs) (string, error) {
		if in.Op == opCreate {
			return "Create the schedule " + quoted(valueOf(in.Name)) + timingOf(&in), nil
		}
		held, err := deps.Schedules.Get(ctx, schedules.GetRequest{ID: in.ScheduleID})
		if err != nil {
			return "", err
		}
		name := quoted(held.Schedule.Name)
		switch in.Op {
		case opUpdate:
			return "Change the schedule " + name + timingOf(&in), nil
		case opDelete:
			return "Delete the schedule " + name, nil
		case opEnable:
			return "Arm the schedule " + name, nil
		case opDisable:
			return "Disarm the schedule " + name + " without deleting it", nil
		default:
			return "Start the run of the schedule " + name + " now", nil
		}
	})
	return riskedBy(tool, map[string]Risk{
		opList: RiskRead, opGet: RiskRead, opCreate: RiskWrite, opUpdate: RiskWrite, opEnable: RiskWrite, opDisable: RiskWrite,
		opRunNow: RiskWrite,
	})
}

func timingOf(in *schedulesArgs) string {
	switch {
	case in.Cron != nil:
		return ", on " + *in.Cron + " UTC"
	case in.IntervalMinutes != nil:
		return ", every " + plural(*in.IntervalMinutes, "minute")
	default:
		return ""
	}
}

func (deps Deps) manageSchedules(ctx context.Context, b Binding, in *schedulesArgs) (any, error) {
	switch in.Op {
	case opList:
		return fitted(deps, in.Limit, func(limit int) (paging.List[schedules.Schedule], error) {
			return deps.Schedules.List(ctx, schedules.ListRequest{
				ListRequest: dto.ListRequest{Cursor: in.Cursor, Limit: limit}, SiteID: b.SiteID, Enabled: in.Enabled,
			})
		})
	case opGet:
		return deps.Schedules.Get(ctx, schedules.GetRequest{ID: in.ScheduleID})
	case opCreate:
		return deps.Schedules.Create(ctx, schedules.CreateRequest{
			SiteID: b.SiteID, Name: valueOf(in.Name), Cron: valueOf(in.Cron), IntervalMinutes: valueOf(in.IntervalMinutes),
			EntityID: valueOf(in.EntityID), Status: valueOf(in.Status), Limit: valueOf(in.MaxPages), TemplateID: valueOf(in.TemplateID),
			Steps: in.Steps, PublishMode: valueOf(in.PublishMode), MaxUSD: valueOf(in.MaxUSD), MaxTokens: valueOf(in.MaxTokens),
			Enabled: valueOf(in.Enabled),
		})
	case opUpdate:
		return deps.Schedules.Update(ctx, schedules.UpdateRequest{
			ID: in.ScheduleID, Name: in.Name, Cron: in.Cron, IntervalMinutes: in.IntervalMinutes, EntityID: in.EntityID,
			Status: in.Status, Limit: in.MaxPages, TemplateID: in.TemplateID, Steps: in.Steps, PublishMode: in.PublishMode,
			MaxUSD: in.MaxUSD, MaxTokens: in.MaxTokens,
		})
	case opDelete:
		return deps.Schedules.Delete(ctx, schedules.DeleteRequest{ID: in.ScheduleID})
	case opEnable:
		return deps.Schedules.Enable(ctx, schedules.EnableRequest{ID: in.ScheduleID})
	case opDisable:
		return deps.Schedules.Disable(ctx, schedules.DisableRequest{ID: in.ScheduleID})
	case opRunNow:
		return deps.Schedules.RunNow(ctx, schedules.RunNowRequest{ID: in.ScheduleID})
	default:
		return nil, errors.New(errors.Invalid, "schedules takes the op list, get, create, update, delete, enable, disable or run_now").
			WithDetail("field", "op")
	}
}
