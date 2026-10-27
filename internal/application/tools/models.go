package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/models"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	modelsName   = "models"
	viewProfiles = "profiles"
	viewUsage    = "usage"
)

type modelsArgs struct {
	View           string `json:"view" enum:"list,profiles,usage" description:"list the models Postulator can call and their prices; profiles the model bound to each job on this site; usage the spend of a run or a conversation"`
	RunID          string `json:"runId,omitempty" description:"usage: only this run's spend"`
	ConversationID string `json:"conversationId,omitempty" description:"usage: only this conversation's spend"`
}

type ModelsResponse struct {
	Models   *models.ListModelsResponse   `json:"models,omitempty"`
	Profiles *models.GetProfilesResponse  `json:"profiles,omitempty"`
	Usage    *models.UsageSummaryResponse `json:"usage,omitempty"`
}

func modelsTool(deps Deps) Tool {
	return NewTool(Def{
		Name:        modelsName,
		Description: "Read the language models, the model each job uses on this site, and what a run or a conversation spent.",
		Risk:        RiskRead,
	}, func(ctx context.Context, b Binding, in modelsArgs) (ModelsResponse, error) {
		switch in.View {
		case viewList:
			listed, err := deps.Models.ListModels(ctx, models.ListModelsRequest{})
			return ModelsResponse{Models: &listed}, err
		case viewProfiles:
			if err := requireSite(ctx, b); err != nil {
				return ModelsResponse{}, err
			}
			bound, err := deps.Models.GetProfiles(ctx, models.GetProfilesRequest{SiteID: b.SiteID})
			return ModelsResponse{Profiles: &bound}, err
		case viewUsage:
			spent, err := deps.Models.UsageSummary(ctx, models.UsageSummaryRequest{RunID: in.RunID, ConversationID: in.ConversationID})
			return ModelsResponse{Usage: &spent}, err
		default:
			return ModelsResponse{}, errors.New(errors.Invalid, "models takes the view list, profiles or usage").WithDetail("field", "view")
		}
	})
}
