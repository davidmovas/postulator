package steps

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/run"
)

func recordWritten(ctx context.Context, deps Deps, sc *run.StepContext, pageID, body, title string, seoApplied []string,
	kept keptFields) error {
	now := deps.now()
	if err := recordBody(ctx, deps, pageID, body); err != nil {
		return err
	}
	if deps.Fields == nil {
		return nil
	}
	if title != "" {
		if err := deps.Fields.Record(ctx, pageID, field.Title, title, sc.Run.ID, now); err != nil {
			return err
		}
	}
	if len(seoApplied) == 0 {
		return nil
	}
	meta, found, err := decodeArtifact[Meta](sc, run.ArtifactMeta)
	if err != nil || !found {
		return err
	}
	for key, value := range map[field.Key]string{
		field.SEOTitle: meta.Title, field.SEODescription: meta.Description, field.SEOCanonical: meta.Canonical,
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
