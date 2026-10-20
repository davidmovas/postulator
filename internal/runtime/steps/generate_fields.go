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
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	NameGenerateFields = string(run.StepGenerateFields)

	CodeFieldNotWritten = "field_not_written"

	fieldsTokens    = 1024
	variationTokens = 256
	imageTokens     = 64
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

func writtenField(sc *run.StepContext, key field.Key) (value string, written bool, err error) {
	fields, _, err := decodeArtifact[Fields](sc, run.ArtifactFields)
	if err != nil {
		return "", false, err
	}
	value, written = fields.Value(key)
	return value, written, nil
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
	reasonNoShort     = "the page's template writes no short description"
)

func GenerateFields(deps Deps) run.StepDef {
	return run.StepDef{
		Name:      NameGenerateFields,
		Role:      domainllm.RoleEditor,
		Produces:  []run.ArtifactKind{run.ArtifactFields},
		Retry:     run.RetryPolicy{Max: 3},
		Timeout:   editorTimeout,
		Price:     run.Price{OutputTokens: fieldsTokens, Input: packInput(deps, domainllm.RoleEditor, fieldsKey), Applies: fieldsApply(deps)},
		Preflight: inputPreflight(deps, domainllm.RoleEditor, fieldsKey),
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
			room := fieldsRoom(asked)
			seen, err := deps.fitted(ctx, sc, call, asked[0].Scope.Key, nil, room, func() (string, string, error) {
				return prompts.Render(NameGenerateFields, prompt)
			}, schemaText[fieldsAnswer]())
			if err != nil {
				return run.Result{}, err
			}
			prompt.Picture = picture(seen.Fitted, contextpack.SectionTargets, contextpack.SectionRules)
			prompt.Asked = askedFields(seen, asked, sc.Spec.Product)

			request, err := stepRequest(sc, call, prompt, room)
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

func fieldsKey(page pagemap.Page) field.Key {
	if page.WPType == pagemap.WPProduct {
		return field.ProductShortDescription
	}
	return field.Excerpt
}

type askedScope struct {
	Key   field.Key
	Name  string
	Scope field.Scope
}

func fieldsRoom(asked []askedScope) int {
	room := fieldsTokens
	for i := range asked {
		switch asked[i].Key.Kind() {
		case field.VariationDescription:
			room += variationTokens
		case field.ImageAlt, field.ImageTitle:
			room += imageTokens
		}
	}
	return room
}

var instancedKinds = []field.Key{field.VariationDescription, field.ImageAlt, field.ImageTitle}

func instancesToWrite(ctx context.Context, deps Deps, sc *run.StepContext, set pagetype.Set, kind pagetype.Type) ([]askedScope, []FieldSkip, error) {
	scopes := make(map[field.Key]field.Scope)
	for _, instanced := range instancedKinds {
		if scope, _ := set.Scope(kind.ID, instanced); scope.Enabled && scope.AIWrites {
			scopes[instanced] = scope
		}
	}
	if len(scopes) == 0 || deps.Fields == nil {
		return nil, nil, nil
	}
	values, err := deps.Fields.ListForPage(ctx, sc.Page.ID)
	if err != nil {
		return nil, nil, err
	}
	candidates := make([]askedScope, 0)
	if scope, enabled := scopes[field.VariationDescription]; enabled {
		for _, variation := range contextpack.Variations(values) {
			named := strings.Join(variation.Options, ", ")
			if named == "" {
				named = variation.Key.Instance()
			}
			candidates = append(candidates, askedScope{
				Key: variation.Key, Name: contextpack.FieldName(field.VariationDescription) + ", " + named, Scope: scope,
			})
		}
	}
	images := make([]askedScope, 0)
	for i := range values {
		instanced := values[i].Key.Kind()
		if scope, enabled := scopes[instanced]; enabled && (instanced == field.ImageAlt || instanced == field.ImageTitle) {
			images = append(images, askedScope{
				Key: values[i].Key, Name: contextpack.FieldName(instanced) + ", image " + values[i].Key.Instance(), Scope: scope,
			})
		}
	}
	slices.SortFunc(images, func(a, b askedScope) int { return strings.Compare(string(a.Key), string(b.Key)) })
	candidates = append(candidates, images...)

	keys := make([]field.Key, 0, len(candidates))
	for i := range candidates {
		keys = append(keys, candidates[i].Key)
	}
	kept, err := deps.keptFieldsOf(ctx, sc, sc.Page.WPID != nil, nil, keys...)
	if err != nil {
		return nil, nil, err
	}
	asked := make([]askedScope, 0, len(candidates))
	skipped := make([]FieldSkip, 0)
	for i := range candidates {
		if kept[candidates[i].Key] {
			skipped = append(skipped, FieldSkip{Key: string(candidates[i].Key), Reason: reasonKept})
			continue
		}
		asked = append(asked, candidates[i])
	}
	return asked, skipped, nil
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
		kept, err := deps.keptFieldsOf(ctx, sc, sc.Page.WPID != nil, nil, key)
		if err != nil {
			return nil, nil, err
		}
		if kept[key] {
			skipped = append(skipped, FieldSkip{Key: string(key), Reason: reasonKept})
			continue
		}
		asked = append(asked, askedScope{Key: key, Scope: scope})
	}
	if sc.Page.WPType == pagemap.WPProduct {
		instances, kept, err := instancesToWrite(ctx, deps, sc, set, kind)
		if err != nil {
			return nil, nil, err
		}
		asked, skipped = append(asked, instances...), append(skipped, kept...)
	}
	return asked, skipped, nil
}

func otherTextFields(page pagemap.Page) []field.Key {
	taken := []field.Key{
		field.Title, field.Body, field.ProductDescription, field.SEOTitle, field.SEODescription, field.SEOCanonical,
	}
	product := page.WPType == pagemap.WPProduct
	out := make([]field.Key, 0)
	for _, spec := range field.Catalog() {
		switch {
		case !spec.Text || spec.Denied || spec.Instanced || field.CheckScope(spec.Key) != nil, slices.Contains(taken, spec.Key):
		case spec.Key == field.Excerpt && product, spec.Key == field.ProductShortDescription && !product:
		default:
			out = append(out, spec.Key)
		}
	}
	return out
}

func writerFor(ctx context.Context, deps Deps, sc *run.StepContext, key field.Key) (string, error) {
	if key == field.ProductShortDescription && sc.Page.WPType == pagemap.WPProduct {
		if !content.WritesShortDescription(sc.Spec.Product) {
			return reasonNoShort, nil
		}
		return "", nil
	}
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
		sc := &run.StepContext{Run: run.Run{SiteID: target.Page.SiteID}, Page: target.Page, Spec: target.Spec}
		asked, _, err := fieldsToWrite(ctx, deps, sc)
		return len(asked) > 0, err
	}
}

func askedFields(seen sitemodel.Picture, asked []askedScope, product *template.Product) []askedField {
	out := make([]askedField, 0, len(asked))
	for i := range asked {
		wanted := &asked[i]
		rules := contextpack.FieldRules(seen.Facts, wanted.Scope)
		if wanted.Key == field.ProductShortDescription && product != nil {
			rules = append(rules, content.ShortDescriptionRules(*product)...)
		}
		name := wanted.Name
		if name == "" {
			name = contextpack.FieldName(wanted.Key)
		}
		out = append(out, askedField{Key: string(wanted.Key), Name: name, Rules: rules})
	}
	return out
}

func settledValue(key field.Key, value string) string {
	if key == field.ProductShortDescription || key.Kind() == field.VariationDescription {
		return content.PlainParagraphs(value)
	}
	return value
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
		settled.Values = append(settled.Values, FieldValue{
			Key: string(wanted.Key), Value: settledValue(wanted.Key, clip(value, wanted.Scope.MaxLength)),
		})
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
