package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/content"
)

const pagesJudgeName = "pages_judge"

func pagesJudge(deps Deps) Tool {
	return describing(newSiteTool(Def{
		Name:        pagesJudgeName,
		Description: "Score the live page against its template and graph with one model call, and say what is wrong.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, b Binding, in content.JudgeRequest) (content.JudgeResponse, error) {
		page, err := deps.ownPage(ctx, b, in.PageID)
		if err != nil {
			return content.JudgeResponse{}, err
		}
		return deps.Content.Judge(ctx, content.JudgeRequest{PageID: page.ID})
	}), func(ctx context.Context, b Binding, in content.JudgeRequest) (string, error) {
		page, err := deps.ownPage(ctx, b, in.PageID)
		if err != nil {
			return "", err
		}
		return "Grade " + page.Path + " against its template and the graph with one model call", nil
	})
}
