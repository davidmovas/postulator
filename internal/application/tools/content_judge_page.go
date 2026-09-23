package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/content"
)

const contentJudgePageName = "content_judge_page"

func contentJudgePage(deps Deps) Tool {
	return NewTool(Def{
		Name:        contentJudgePageName,
		Description: "Score the page WordPress is serving right now against its template and its graph, and say what is wrong with it.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in content.JudgeRequest) (content.JudgeResponse, error) {
		return deps.Content.Judge(ctx, in)
	})
}
