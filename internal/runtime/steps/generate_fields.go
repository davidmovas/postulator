package steps

import (
	"context"
	"slices"
	"strings"

	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	NameGenerateFields = string(run.StepGenerateFields)

	CodeFieldNotWritten = "field_not_written"

	fieldsTokens = 1024
)

type FieldValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type FieldSkip struct {
	Key    string `json:"key"`
	Reason string `json:"reason"`
}

type Fields struct {
	Values  []FieldValue `json:"values"`
	Skipped []FieldSkip  `json:"skipped"`
}

func (f Fields) Value(key field.Key) (string, bool) {
	for _, value := range f.Values {
		if value.Key == string(key) {
			return value.Value, true
		}
	}
	return "", false
}

type fieldAnswer struct {
	Key   string `json:"key" description:"The key of the field, exactly as the brief names it"`
	Value string `json:"value" description:"The field's text, plain, within the field's length rule"`
}

type fieldsAnswer struct {
	Fields []fieldAnswer `json:"fields" description:"One entry for every field the brief asks for"`
}

type fieldsPrompt struct {
	Page    pagemap.Page
	Picture string
	Asked   []askedField
}

type askedField struct {
	Key   string
	Name  string
	Rules []string
}

const (
	reasonNoWriter    = "nothing on this site writes it yet"
	reasonUnsupported = "the page's post type carries no excerpt"
	reasonKept        = "it holds the owner's own text, and the field's rewrites switch is off"
)

func GenerateFields(deps Deps) run.StepDef {
	return run.StepDef{
		Name:      NameGenerateFields,
		Role:      domainllm.RoleEditor,
		Produces:  []run.ArtifactKind{run.ArtifactFields},
		Retry:     run.RetryPolicy{Max: 3},
		Timeout:   editorTimeout,
		Price:     run.Price{OutputTokens: fieldsTokens, Input: packInput(deps, domainllm.RoleEditor, excerptKey), Applies: fieldsApply(deps)},
		Preflight: inputPreflight(deps, domainllm.RoleEditor, excerptKey),
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			asked, skipped, err := fieldsToWrite(ctx, deps, sc)
			if err != nil {
				return run.Result{}, err
			}
			if len(asked) == 0 {
				return fieldsResult(Fields{Values: []FieldValue{}, Skipped: skipped}, 0,
					"asked for no other field of "+sc.Page.Path)
			}

			call, err := modelFor(ctx, deps, sc, NameGenerateFields, domainllm.RoleEditor)
			if err != nil {
				return run.Result{}, err
			}
			prompt := fieldsPrompt{Page: sc.Page}
			seen, err := deps.fitted(ctx, sc, call, asked[0].Key, nil, fieldsTokens, func() (string, string, error) {
				return prompts.Render(NameGenerateFields, prompt)
			}, schemaText[fieldsAnswer]())
			if err != nil {
				return run.Result{}, err
			}
			prompt.Picture = picture(seen.Fitted, contextpack.SectionTargets, contextpack.SectionRules)
			prompt.Asked = askedFields(seen, asked)

			request, err := stepRequest(sc, call, prompt, fieldsTokens)
			if err != nil {
				return run.Result{}, err
			}
			answer, usage, err := port.Structured[fieldsAnswer](ctx, deps.LLM, request)
			if err != nil {
				return run.Result{}, err
			}
			settled, err := settleFields(answer, asked)
			if err != nil {
				return run.Result{}, err
			}
			settled.Skipped = skipped
			return fieldsResult(settled, usage.Total, "wrote "+plural(len(settled.Values))+" of "+sc.Page.Path)
		},
	}
}

func excerptKey(pagemap.Page) field.Key {
	return field.Excerpt
}

type askedScope struct {
	Key   field.Key
	Scope field.Scope
}

func fieldsToWrite(ctx context.Context, deps Deps, sc *run.StepContext) ([]askedScope, []FieldSkip, error) {
	set, err := deps.Types.Set(ctx, sc.Run.SiteID)
	if err != nil {
		return nil, nil, err
	}
	kind := claimedType(set, sc)
	asked := make([]askedScope, 0)
	skipped := make([]FieldSkip, 0)
	candidates := make([]field.Scope, 0)
	for _, key := range otherTextFields(sc.Page) {
		scope, _ := set.Scope(kind.ID, key)
		candidates = append(candidates, scope)
	}
	candidates = append(candidates, set.Custom(kind.ID)...)
	for i := range candidates {
		scope, key := candidates[i], candidates[i].Key
		if !scope.Enabled || !scope.AIWrites {
			continue
		}
		reason, err := writerFor(ctx, deps, sc, key)
		if err != nil {
			return nil, nil, err
		}
		if reason != "" {
			skipped = append(skipped, FieldSkip{Key: string(key), Reason: reason})
			continue
		}
		kept, err := deps.keptFieldsOf(ctx, sc, sc.Page.WPID != nil, key)
		if err != nil {
			return nil, nil, err
		}
		if kept[key] {
			skipped = append(skipped, FieldSkip{Key: string(key), Reason: reasonKept})
			continue
		}
		asked = append(asked, askedScope{Key: key, Scope: scope})
	}
	return asked, skipped, nil
}

func otherTextFields(page pagemap.Page) []field.Key {
	taken := []field.Key{
		field.Title, field.Body, field.ProductDescription, field.ProductShortDescription, field.SEOTitle, field.SEODescription,
		field.SEOCanonical,
	}
	out := make([]field.Key, 0)
	for _, spec := range field.Catalog() {
		switch {
		case !spec.Text || spec.Denied || spec.Instanced || field.CheckScope(spec.Key) != nil, slices.Contains(taken, spec.Key):
		case spec.Key == field.Excerpt && page.WPType == pagemap.WPProduct:
		default:
			out = append(out, spec.Key)
		}
	}
	return out
}

func writerFor(ctx context.Context, deps Deps, sc *run.StepContext, key field.Key) (string, error) {
	if key != field.Excerpt || sc.Page.WPType == pagemap.WPProduct {
		return reasonNoWriter, nil
	}
	client, err := clientFor(ctx, deps, sc.Run.SiteID)
	if err != nil {
		return "", err
	}
	types, err := client.PostTypes(ctx)
	if err != nil {
		return "", err
	}
	for _, postType := range types {
		if postType.Slug == string(sc.Page.WPType) && postType.Has("excerpt") {
			return "", nil
		}
	}
	return reasonUnsupported, nil
}

func fieldsApply(deps Deps) func(context.Context, run.Target) (bool, error) {
	if deps.Types == nil || deps.WordPress == nil {
		return nil
	}
	return func(ctx context.Context, target run.Target) (bool, error) {
		sc := &run.StepContext{Run: run.Run{SiteID: target.Page.SiteID}, Page: target.Page}
		asked, _, err := fieldsToWrite(ctx, deps, sc)
		return len(asked) > 0, err
	}
}

func askedFields(seen sitemodel.Picture, asked []askedScope) []askedField {
	out := make([]askedField, 0, len(asked))
	for i := range asked {
		wanted := &asked[i]
		out = append(out, askedField{
			Key: string(wanted.Key), Name: contextpack.FieldName(wanted.Key), Rules: contextpack.FieldRules(seen.Facts, wanted.Scope),
		})
	}
	return out
}

func settleFields(answer fieldsAnswer, asked []askedScope) (Fields, error) {
	settled := Fields{Values: make([]FieldValue, 0, len(asked))}
	for i := range asked {
		wanted := &asked[i]
		value := ""
		for _, given := range answer.Fields {
			if given.Key == string(wanted.Key) {
				value = strings.TrimSpace(given.Value)
				break
			}
		}
		if value == "" {
			return Fields{}, errors.New(errors.External, "the model left out a field it was asked for").
				WithDetail("reason", content.ReasonIncompleteAnswer).WithDetail("field", string(wanted.Key))
		}
		settled.Values = append(settled.Values, FieldValue{Key: string(wanted.Key), Value: clip(value, wanted.Scope.MaxLength)})
	}
	return settled, nil
}

func fieldsResult(fields Fields, tokens int, message string) (run.Result, error) {
	blob, err := encode(fields, "the fields")
	if err != nil {
		return run.Result{}, err
	}
	return run.Result{
		Artifacts: []run.Artifact{{Kind: run.ArtifactFields, Blob: blob}},
		Tokens:    tokens,
		Message:   message,
	}, nil
}
