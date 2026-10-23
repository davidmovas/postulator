package run

import (
	"slices"

	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/template"
)

func ScopeRecipe(recipe []template.StepSpec, keys []string) ([]template.StepSpec, error) {
	if len(keys) == 0 {
		return recipe, nil
	}
	scope := make([]field.Key, 0, len(keys))
	for _, key := range keys {
		spec, known := field.Lookup(field.Key(key))
		if !known || !spec.AIWritable() {
			return nil, invalid("a run can be scoped only to fields it may write", "fields").WithDetail("key", key)
		}
		scope = append(scope, field.Key(key))
	}
	slices.Sort(scope)
	scope = slices.Compact(scope)
	if !slices.ContainsFunc(recipe, enabledAs(StepPublish)) {
		return nil, invalid("the run publishes nothing, so it has no fields to scope", "fields")
	}

	listed := make([]string, 0, len(scope))
	for _, key := range scope {
		listed = append(listed, string(key))
	}
	out := slices.Clone(recipe)
	for i := range out {
		if !out[i].Enabled {
			continue
		}
		if !Serves(StepName(out[i].Name), scope) {
			out[i].Enabled = false
			continue
		}
		params := make(map[string]any, len(out[i].Params)+1)
		for name, value := range out[i].Params {
			params[name] = value
		}
		params[ParamFields] = slices.Clone(listed)
		out[i].Params = params
	}
	for _, key := range scope {
		if !slices.ContainsFunc(out, enabledAs(Producer(key, scope))) {
			return nil, invalid("the recipe has no step that writes "+string(key), "fields").WithDetail("key", string(key))
		}
	}
	return out, nil
}

func enabledAs(step StepName) func(template.StepSpec) bool {
	return func(spec template.StepSpec) bool {
		return spec.Enabled && spec.Name == string(step)
	}
}

func WritesBody(scope []field.Key) bool {
	return slices.Contains(scope, field.Body) || slices.Contains(scope, field.ProductDescription)
}

func drawsTheBody(scope []field.Key) bool {
	return WritesBody(scope) || slices.Contains(scope, field.FeaturedImage)
}

func Producer(key field.Key, scope []field.Key) StepName {
	switch key.Kind() {
	case field.Body, field.ProductDescription:
		return StepGenerateBody
	case field.Title:
		if WritesBody(scope) {
			return StepGenerateBody
		}
		return StepGenerateFields
	case field.SEOTitle, field.SEODescription:
		return StepGenerateMeta
	case field.FeaturedImage:
		return StepGenerateImages
	default:
		return StepGenerateFields
	}
}

func Serves(step StepName, scope []field.Key) bool {
	switch step {
	case StepResolveContext, StepGenerateBody, StepGenerateImages:
		return drawsTheBody(scope)
	case StepInsertLinks, StepRepairLinks, StepValidate, StepJudge, StepRelinkNeighbors:
		return WritesBody(scope)
	case StepGenerateMeta, StepGenerateFields:
		return slices.ContainsFunc(scope, func(key field.Key) bool { return Producer(key, scope) == step })
	case StepPublish, StepSyncBack, StepReport:
		return true
	default:
		return true
	}
}

func DraftsOf(scope []field.Key) []ArtifactKind {
	out := make([]ArtifactKind, 0, 3)
	add := func(kinds ...ArtifactKind) {
		for _, kind := range kinds {
			if !slices.Contains(out, kind) {
				out = append(out, kind)
			}
		}
	}
	for _, key := range scope {
		switch Producer(key, scope) {
		case StepGenerateBody:
			add(ArtifactDraft, ArtifactBodyHTML)
		case StepGenerateMeta:
			add(ArtifactMeta)
		case StepGenerateImages:
			add(ArtifactImages)
		default:
			add(ArtifactFields)
		}
	}
	return out
}

func FieldsOf(params map[string]any) []field.Key {
	listed, held := params[ParamFields]
	if !held {
		return nil
	}
	var out []field.Key
	switch keys := listed.(type) {
	case []string:
		for _, key := range keys {
			out = append(out, field.Key(key))
		}
	case []any:
		for _, key := range keys {
			if text, ok := key.(string); ok {
				out = append(out, field.Key(text))
			}
		}
	}
	return out
}

func (d StepDef) RequiresFor(params map[string]any) []ArtifactKind {
	if d.Scoped == nil {
		return d.Requires
	}
	if scope := FieldsOf(params); len(scope) > 0 {
		return d.Scoped(scope)
	}
	return d.Requires
}
