package steps

import (
	"context"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/run"
)

func recordWritten(ctx context.Context, deps Deps, sc *run.StepContext, pageID, title string, seoApplied []string,
	meta *wp.SEOMeta, kept keptFields) error {
	now := deps.now()
	if deps.Fields == nil {
		return nil
	}
	if title != "" {
		if err := deps.Fields.Record(ctx, pageID, field.Title, title, sc.Run.ID, now); err != nil {
			return err
		}
	}
	if len(seoApplied) == 0 || meta == nil {
		return nil
	}
	for key, value := range map[field.Key]string{
		field.SEOTitle: meta.Title, field.SEODescription: meta.Description,
	} {
		if kept[key] {
			continue
		}
		if recordErr := deps.Fields.Record(ctx, pageID, key, value, sc.Run.ID, now); recordErr != nil {
			return recordErr
		}
	}
	return nil
}

func recordBody(ctx context.Context, deps Deps, pageID, body string) error {
	if deps.Bodies == nil {
		return nil
	}
	return deps.Bodies.Record(ctx, pageID, body, deps.now())
}

func recordField(ctx context.Context, deps Deps, sc *run.StepContext, pageID string, key field.Key, value string) error {
	if deps.Fields == nil {
		return nil
	}
	return deps.Fields.Record(ctx, pageID, key, value, sc.Run.ID, deps.now())
}
