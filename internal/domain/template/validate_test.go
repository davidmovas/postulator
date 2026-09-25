package template_test

import (
	"encoding/json"
	stderrors "errors"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

var stamp = time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC)

func ptr(s string) *string {
	return &s
}

func fieldOf(t *testing.T, err error) string {
	t.Helper()
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) {
		t.Fatalf("error %v is not a kernel error", err)
	}
	field, ok := kernel.Details["field"].(string)
	if !ok {
		t.Fatalf("error %v carries no field detail", err)
	}
	return field
}

func validSpec() template.TemplateSpec {
	return template.TemplateSpec{
		Sections: []template.Section{
			{Heading: "Overview", Intent: "Define the topic", TargetWords: 200, Required: true, KeywordRules: template.SectionKeywordRules{Include: []string{}, PrimaryInHeading: true}},
			{Heading: "Details", Intent: "Explain", TargetWords: 400, Required: true, KeywordRules: template.SectionKeywordRules{Include: []string{}}},
		},
		Tone:          "plain",
		Length:        template.Length{Min: 500, Max: 900},
		KeywordRules:  template.KeywordRules{PrimaryInTitle: true, PrimaryInH1: true, PrimaryInFirstParagraph: true, MaxDensity: 0.02},
		LinkRules:     template.LinkRules{UpDepth: 2, DownLinks: true, SiblingMinWeight: 0.5, MaxLinks: 10, MaxPerTarget: 1, ParentLinkWithinParagraphs: 2, ChildrenSection: true},
		MetaRules:     template.MetaRules{TitlePattern: "{primaryKeyword} | {siteName}", DescriptionMax: 155},
		Images:        template.Images{Featured: true, Inline: 1, Source: template.ImagesAI},
		ModelProfiles: map[llm.Role]llm.ModelRef{llm.RoleWriter: {Provider: "openai", Model: "gpt"}},
		Recipe:        []template.StepSpec{{Name: "resolve_context", Enabled: true}, {Name: "generate_body", Enabled: true}},
	}
}

func TestAnUnknownPlaceholderIsRefusedByNameWithTheOnesThatWork(t *testing.T) {
	t.Parallel()

	spec := validSpec()
	spec.Sections[0].Heading = "Best {entity} for {primaryKeyword}"

	err := template.Validate(spec)
	if err == nil {
		t.Fatal("Validate accepted a placeholder nothing fills")
	}
	for _, want := range []string{"{entity}", "{primaryKeyword}", "{entityName}", "{siteName}", "{pageTitle}"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal %q does not name %s", err.Error(), want)
		}
	}
}

func TestValidateSpec(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*template.TemplateSpec)
		field  string
	}{
		{name: "valid", mutate: func(*template.TemplateSpec) {}},
		{name: "no sections", mutate: func(s *template.TemplateSpec) { s.Sections = nil }, field: "sections"},
		{name: "blank heading", mutate: func(s *template.TemplateSpec) { s.Sections[1].Heading = " " }, field: "sections[1].heading"},
		{name: "unknown heading placeholder", mutate: func(s *template.TemplateSpec) { s.Sections[1].Heading = "Best {entity} guide" }, field: "sections[1].heading"},
		{name: "double-braced heading placeholder", mutate: func(s *template.TemplateSpec) { s.Sections[0].Heading = "{{primaryKeyword}} tips" }, field: "sections[0].heading"},
		{name: "unknown intent placeholder", mutate: func(s *template.TemplateSpec) { s.Sections[0].Intent = "Cover {topic}" }, field: "sections[0].intent"},
		{name: "unknown pinned phrase placeholder", mutate: func(s *template.TemplateSpec) { s.Sections[0].KeywordRules.Include = []string{"{keyword} guide"} }, field: "sections[0].keywordRules.include[0]"},
		{name: "unknown title placeholder", mutate: func(s *template.TemplateSpec) { s.MetaRules.TitlePattern = "{title} | {siteName}" }, field: "metaRules.titlePattern"},
		{name: "negative target words", mutate: func(s *template.TemplateSpec) { s.Sections[0].TargetWords = -1 }, field: "sections[0].targetWords"},
		{name: "negative min", mutate: func(s *template.TemplateSpec) { s.Length.Min = -1 }, field: "length.min"},
		{name: "max below min", mutate: func(s *template.TemplateSpec) { s.Length.Max = 100 }, field: "length.max"},
		{name: "density above one", mutate: func(s *template.TemplateSpec) { s.KeywordRules.MaxDensity = 1.5 }, field: "keywordRules.maxDensity"},
		{name: "negative up depth", mutate: func(s *template.TemplateSpec) { s.LinkRules.UpDepth = -1 }, field: "linkRules.upDepth"},
		{name: "sibling weight", mutate: func(s *template.TemplateSpec) { s.LinkRules.SiblingMinWeight = 2 }, field: "linkRules.siblingMinWeight"},
		{name: "max links", mutate: func(s *template.TemplateSpec) { s.LinkRules.MaxLinks = -1 }, field: "linkRules.maxLinks"},
		{name: "max per target", mutate: func(s *template.TemplateSpec) { s.LinkRules.MaxPerTarget = -1 }, field: "linkRules.maxPerTarget"},
		{name: "paragraph window", mutate: func(s *template.TemplateSpec) { s.LinkRules.ParentLinkWithinParagraphs = -1 }, field: "linkRules.parentLinkWithinParagraphs"},
		{name: "description max", mutate: func(s *template.TemplateSpec) { s.MetaRules.DescriptionMax = -1 }, field: "metaRules.descriptionMax"},
		{name: "inline images", mutate: func(s *template.TemplateSpec) { s.Images.Inline = -1 }, field: "images.inline"},
		{name: "unknown image source", mutate: func(s *template.TemplateSpec) { s.Images.Source = "camera" }, field: "images.source"},
		{name: "images without a source", mutate: func(s *template.TemplateSpec) { s.Images.Source = "" }, field: "images.source"},
		{name: "no images no source is fine", mutate: func(s *template.TemplateSpec) { s.Images = template.Images{} }},
		{name: "unknown role", mutate: func(s *template.TemplateSpec) {
			s.ModelProfiles = map[llm.Role]llm.ModelRef{"painter": {Provider: "a", Model: "b"}}
		}, field: "modelProfiles.painter"},
		{name: "incomplete ref", mutate: func(s *template.TemplateSpec) {
			s.ModelProfiles = map[llm.Role]llm.ModelRef{llm.RoleJudge: {Model: "b"}}
		}, field: "modelProfiles.judge"},
		{name: "blank step", mutate: func(s *template.TemplateSpec) { s.Recipe[1].Name = "" }, field: "recipe[1].name"},
		{name: "repeated step", mutate: func(s *template.TemplateSpec) { s.Recipe[1].Name = "resolve_context" }, field: "recipe[1].name"},
		{name: "empty recipe is allowed", mutate: func(s *template.TemplateSpec) { s.Recipe = nil }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			spec := validSpec()
			tc.mutate(&spec)
			err := template.Validate(spec)
			if tc.field == "" {
				if err != nil {
					t.Fatalf("Validate: %v", err)
				}
				return
			}
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("code = %q, want INVALID", errors.CodeOf(err))
			}
			if got := fieldOf(t, err); got != tc.field {
				t.Errorf("field = %q, want %q", got, tc.field)
			}
		})
	}
}

func TestTemplateValidate(t *testing.T) {
	t.Parallel()

	base := template.Template{ID: "t1", Scope: template.ScopeGlobal, Name: "Hub", PageKind: "hub", Version: 1, Spec: validSpec(), CreatedAt: stamp, UpdatedAt: stamp}

	cases := []struct {
		name   string
		mutate func(*template.Template)
		field  string
	}{
		{name: "valid global", mutate: func(*template.Template) {}},
		{name: "valid site", mutate: func(x *template.Template) { x.Scope = template.ScopeSite; x.SiteID = ptr("s1") }},
		{name: "no id", mutate: func(x *template.Template) { x.ID = "" }, field: "id"},
		{name: "unknown scope", mutate: func(x *template.Template) { x.Scope = "galaxy" }, field: "scope"},
		{name: "global with site", mutate: func(x *template.Template) { x.SiteID = ptr("s1") }, field: "siteId"},
		{name: "site without site", mutate: func(x *template.Template) { x.Scope = template.ScopeSite }, field: "siteId"},
		{name: "blank name", mutate: func(x *template.Template) { x.Name = " " }, field: "name"},
		{name: "blank kind", mutate: func(x *template.Template) { x.PageKind = "" }, field: "pageKind"},
		{name: "zero version", mutate: func(x *template.Template) { x.Version = 0 }, field: "version"},
		{name: "bad spec", mutate: func(x *template.Template) { x.Spec.Sections = nil }, field: "sections"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			x := base
			tc.mutate(&x)
			err := x.Validate()
			if tc.field == "" {
				if err != nil {
					t.Fatalf("Validate: %v", err)
				}
				return
			}
			if got := fieldOf(t, err); got != tc.field {
				t.Errorf("field = %q, want %q (err %v)", got, tc.field, err)
			}
		})
	}
}

func TestLinkPolicyValidate(t *testing.T) {
	t.Parallel()

	base := template.LinkPolicy{ID: "p1", Scope: template.ScopeSite, SiteID: ptr("s1"), Name: "Strict", Rules: validSpec().LinkRules, ForbidExternal: true, ForbidSelf: true, AnchorStrategy: template.AnchorPreferUser, CreatedAt: stamp, UpdatedAt: stamp}

	cases := []struct {
		name   string
		mutate func(*template.LinkPolicy)
		field  string
	}{
		{name: "valid", mutate: func(*template.LinkPolicy) {}},
		{name: "no id", mutate: func(p *template.LinkPolicy) { p.ID = "" }, field: "id"},
		{name: "site without site", mutate: func(p *template.LinkPolicy) { p.SiteID = nil }, field: "siteId"},
		{name: "global with site", mutate: func(p *template.LinkPolicy) { p.Scope = template.ScopeGlobal }, field: "siteId"},
		{name: "blank name", mutate: func(p *template.LinkPolicy) { p.Name = "" }, field: "name"},
		{name: "bad rules", mutate: func(p *template.LinkPolicy) { p.Rules.MaxLinks = -3 }, field: "linkRules.maxLinks"},
		{name: "unknown strategy", mutate: func(p *template.LinkPolicy) { p.AnchorStrategy = "random" }, field: "anchorStrategy"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := base
			tc.mutate(&p)
			err := p.Validate()
			if tc.field == "" {
				if err != nil {
					t.Fatalf("Validate: %v", err)
				}
				return
			}
			if got := fieldOf(t, err); got != tc.field {
				t.Errorf("field = %q, want %q (err %v)", got, tc.field, err)
			}
		})
	}
}

func TestOverrideValidate(t *testing.T) {
	t.Parallel()

	base := template.Override{ID: "o1", TemplateID: "t1", Scope: template.OverrideSite, TargetID: "s1", Patch: json.RawMessage(`{"tone":"warm"}`), CreatedAt: stamp, UpdatedAt: stamp}

	cases := []struct {
		name   string
		mutate func(*template.Override)
		field  string
	}{
		{name: "valid", mutate: func(*template.Override) {}},
		{name: "no id", mutate: func(o *template.Override) { o.ID = "" }, field: "id"},
		{name: "no template", mutate: func(o *template.Override) { o.TemplateID = "" }, field: "templateId"},
		{name: "unknown scope", mutate: func(o *template.Override) { o.Scope = "galaxy" }, field: "scope"},
		{name: "no target", mutate: func(o *template.Override) { o.TargetID = "" }, field: "targetId"},
		{name: "patch is not json", mutate: func(o *template.Override) { o.Patch = json.RawMessage(`{`) }, field: "patch"},
		{name: "patch is not an object", mutate: func(o *template.Override) { o.Patch = json.RawMessage(`[1]`) }, field: "patch"},
		{name: "empty patch", mutate: func(o *template.Override) { o.Patch = nil }, field: "patch"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			o := base
			tc.mutate(&o)
			err := o.Validate()
			if tc.field == "" {
				if err != nil {
					t.Fatalf("Validate: %v", err)
				}
				return
			}
			if got := fieldOf(t, err); got != tc.field {
				t.Errorf("field = %q, want %q (err %v)", got, tc.field, err)
			}
		})
	}
}

func TestEnums(t *testing.T) {
	t.Parallel()

	if !template.ScopeGlobal.Valid() || !template.ScopeSite.Valid() || template.Scope("x").Valid() {
		t.Error("scope validity is wrong")
	}
	if !template.ImagesAI.Valid() || !template.ImagesWPMedia.Valid() || !template.ImagesLocal.Valid() || template.ImageSource("x").Valid() {
		t.Error("image source validity is wrong")
	}
	if !template.AnchorPreferUser.Valid() || !template.AnchorRotate.Valid() || template.AnchorStrategy("x").Valid() {
		t.Error("anchor strategy validity is wrong")
	}
	if !template.OverrideSite.Valid() || !template.OverridePage.Valid() || template.OverrideScope("x").Valid() {
		t.Error("override scope validity is wrong")
	}
	if !template.SortCreatedAt.Valid() || !template.SortName.Valid() || template.Sort("x").Valid() {
		t.Error("sort validity is wrong")
	}
}

func TestSpecJSONIsCamelCase(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(validSpec())
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	for _, key := range []string{`"sections"`, `"targetWords"`, `"keywordRules"`, `"primaryInHeading"`, `"linkRules"`, `"siblingMinWeight"`, `"metaRules"`, `"descriptionMax"`, `"modelProfiles"`, `"recipe"`} {
		if !strings.Contains(string(encoded), key) {
			t.Errorf("encoded spec lacks %s: %s", key, encoded)
		}
	}
}

func TestImagesCountWhatThePageAsksFor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		images template.Images
		want   int
	}{
		{name: "none", images: template.Images{}, want: 0},
		{name: "a featured image", images: template.Images{Featured: true, Source: template.ImagesAI}, want: 1},
		{name: "inline images", images: template.Images{Inline: 2, Source: template.ImagesAI}, want: 2},
		{name: "both", images: template.Images{Featured: true, Inline: 3, Source: template.ImagesWPMedia}, want: 4},
		{name: "a negative count asks for none", images: template.Images{Inline: -2}, want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.images.Wanted(); got != tc.want {
				t.Fatalf("Wanted() = %d, want %d", got, tc.want)
			}
		})
	}
}
