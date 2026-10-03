package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/content"
)

const contentJudgePageName = "content_judge_page"

func contentJudgePage(deps Deps) Tool {
	return NewTool(Def{
		Name:        contentJudgePageName,
		Description: "Score the live page against its template and graph, and say what is wrong.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in content.JudgeRequest) (content.JudgeResponse, error) {
		return deps.Content.Judge(ctx, in)
	})
}
