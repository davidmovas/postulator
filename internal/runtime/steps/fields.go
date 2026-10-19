package steps

import (
	"context"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/run"
)

type keptFields map[field.Key]bool

func claimedType(set pagetype.Set, sc *run.StepContext) pagetype.Type {
	if sc.Type.ID != "" {
		return sc.Type
	}
	return set.TypeOf(sc.Page)
}

func (k keptFields) all(keys ...field.Key) bool {
	for _, key := range keys {
		if !k[key] {
			return false
		}
	}
	return len(keys) > 0
}

func (d Deps) keptFieldsOf(ctx context.Context, sc *run.StepContext, existing bool, keys ...field.Key) (keptFields, error) {
	kept := make(keptFields, len(keys))
	set, err := d.Types.Set(ctx, sc.Run.SiteID)
	if err != nil {
		return nil, err
	}
	values := make(map[field.Key]field.Value)
	if d.Fields != nil && existing {
		held, listErr := d.Fields.ListForPage(ctx, sc.Page.ID)
		if listErr != nil {
			return nil, listErr
		}
		for i := range held {
			values[held[i].Key] = held[i]
		}
	}
	kind := claimedType(set, sc)
	for _, key := range keys {
		scope, _ := set.Scope(kind.ID, key)
		value := values[key]
		ours := value.WrittenRunID != nil && value.CurrentHash == value.BaseHash
		rewrites := scope.Rewrites || sc.Rewrites(sc.Page.ID)
		switch {
		case !scope.Enabled || !scope.AIWrites:
			kept[key] = existing || key != field.Title
		case existing && !rewrites && strings.TrimSpace(value.Current) != "" && !ours:
			kept[key] = true
		}
	}
	return kept, nil
}
