package tools

import (
	"encoding/json"

	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type modelProfileArgs struct {
	Role     string `json:"role" enum:"writer,editor,linker,judge,chat,image,titler" description:"Model job"`
	Provider string `json:"provider" description:"Provider, openai"`
	Model    string `json:"model" description:"Name from the models list"`
}

type stepArgs struct {
	Name        string `json:"name" enum:"resolve_context,generate_body,generate_meta,generate_fields,insert_links,repair_links,generate_images,validate,judge,publish,relink_neighbors,sync_back,report" description:"Step name"`
	Enabled     bool   `json:"enabled,omitempty" description:"Run it; false keeps it skipped"`
	AllowErrors *bool  `json:"allowErrors,omitempty" description:"validate only: go on despite faults"`
	Iterations  *int   `json:"iterations,omitempty" minimum:"1" description:"repair_links only: max passes"`
}

type sectionKeywordRulesArgs struct {
	Include          []string `json:"include,omitempty" description:"Phrases it must use"`
	PrimaryInHeading bool     `json:"primaryInHeading,omitempty" description:"Primary keyword in the heading"`
}

func (a sectionKeywordRulesArgs) rules() template.SectionKeywordRules {
	return template.SectionKeywordRules{Include: a.Include, PrimaryInHeading: a.PrimaryInHeading}
}

type sectionArgs struct {
	Heading      string                   `json:"heading" description:"May use {primaryKeyword}, {entityName}, {siteName}, {pageTitle}; no other braces"`
	Intent       string                   `json:"intent,omitempty" description:"What it covers"`
	TargetWords  int                      `json:"targetWords,omitempty" minimum:"0" description:"Approximate words"`
	Required     bool                     `json:"required,omitempty" description:"Page invalid without it"`
	KeywordRules *sectionKeywordRulesArgs `json:"keywordRules,omitempty" description:"Section keyword rules"`
}

func (a sectionArgs) section() template.Section {
	built := template.Section{
		Heading: a.Heading, Intent: a.Intent, TargetWords: a.TargetWords, Required: a.Required,
	}
	if a.KeywordRules != nil {
		built.KeywordRules = a.KeywordRules.rules()
	}
	return built
}

func sectionsOf(listed []sectionArgs) []template.Section {
	if len(listed) == 0 {
		return nil
	}

	sections := make([]template.Section, 0, len(listed))
	for _, section := range listed {
		sections = append(sections, section.section())
	}
	return sections
}

type lengthArgs struct {
	Min int `json:"min,omitempty" minimum:"0" description:"Fewest words"`
	Max int `json:"max,omitempty" minimum:"0" description:"Most words"`
}

func (a lengthArgs) length() template.Length {
	return template.Length{Min: a.Min, Max: a.Max}
}

type keywordRulesArgs struct {
	PrimaryInTitle          bool    `json:"primaryInTitle,omitempty" description:"Primary keyword in the title"`
	PrimaryInH1             bool    `json:"primaryInH1,omitempty" description:"Primary keyword in the H1"`
	PrimaryInFirstParagraph bool    `json:"primaryInFirstParagraph,omitempty" description:"Primary keyword in the first paragraph"`
	MaxDensity              float64 `json:"maxDensity,omitempty" minimum:"0" maximum:"1" description:"Max share of words for the primary keyword"`
	RequiredKeywords        *int    `json:"requiredKeywords,omitempty" minimum:"0" description:"Top keywords the body must use; default all"`
}

func (a keywordRulesArgs) rules() template.KeywordRules {
	return template.KeywordRules{
		PrimaryInTitle:          a.PrimaryInTitle,
		PrimaryInH1:             a.PrimaryInH1,
		PrimaryInFirstParagraph: a.PrimaryInFirstParagraph,
		MaxDensity:              a.MaxDensity,
		RequiredKeywords:        a.RequiredKeywords,
	}
}

type metaRulesArgs struct {
	TitlePattern   string `json:"titlePattern,omitempty" description:"e.g. {primaryKeyword} | {siteName}; default the page title"`
	DescriptionMax int    `json:"descriptionMax,omitempty" minimum:"0" description:"Max SEO description chars"`
}

func (a metaRulesArgs) rules() template.MetaRules {
	return template.MetaRules{TitlePattern: a.TitlePattern, DescriptionMax: a.DescriptionMax}
}

type imagesArgs struct {
	Featured bool                 `json:"featured,omitempty" description:"Add a featured image"`
	Inline   int                  `json:"inline,omitempty" minimum:"0" description:"Images in the body"`
	Source   template.ImageSource `json:"source,omitempty" enum:"ai,wpmedia,local" description:"Image origin; required with any image"`
}

func (a imagesArgs) images() template.Images {
	return template.Images{Featured: a.Featured, Inline: a.Inline, Source: a.Source}
}

type productShortDescriptionArgs struct {
	Enabled        bool   `json:"enabled,omitempty" description:"Write it"`
	Intent         string `json:"intent,omitempty" description:"What it says"`
	TargetWords    int    `json:"targetWords,omitempty" minimum:"0" description:"Approximate words"`
	PrimaryKeyword bool   `json:"primaryKeyword,omitempty" description:"Must contain the primary keyword"`
}

type productArgs struct {
	ShortDescription *productShortDescriptionArgs `json:"shortDescription,omitempty" description:"Text shown beside the price"`
}

func (a productArgs) product() *template.Product {
	built := &template.Product{}
	if a.ShortDescription != nil {
		built.ShortDescription = template.ProductShortDescription{
			Enabled: a.ShortDescription.Enabled, Intent: a.ShortDescription.Intent,
			TargetWords: a.ShortDescription.TargetWords, PrimaryKeyword: a.ShortDescription.PrimaryKeyword,
		}
	}
	return built
}

type templateSpecArgs struct {
	Sections      []sectionArgs      `json:"sections,omitempty" description:"Sections in order; a specification holds at least one"`
	Tone          *string            `json:"tone,omitempty" description:"Voice, one or two sentences"`
	Length        *lengthArgs        `json:"length,omitempty" description:"Word count window"`
	KeywordRules  *keywordRulesArgs  `json:"keywordRules,omitempty" description:"Keyword placement and density"`
	MetaRules     *metaRulesArgs     `json:"metaRules,omitempty" description:"SEO title and description"`
	Images        *imagesArgs        `json:"images,omitempty" description:"Images; default none"`
	Product       *productArgs       `json:"product,omitempty" description:"Product outputs; omit for pages"`
	ModelProfiles []modelProfileArgs `json:"modelProfiles,omitempty" description:"Model per job; default the site's"`
	Recipe        []stepArgs         `json:"recipe,omitempty" description:"Steps in order; default recipe"`
}

func (a templateSpecArgs) spec() template.TemplateSpec {
	built := template.TemplateSpec{
		Sections:      sectionsOf(a.Sections),
		Tone:          valueOf(a.Tone),
		ModelProfiles: profilesOf(a.ModelProfiles),
		Recipe:        recipeOfSteps(a.Recipe),
	}
	if a.Length != nil {
		built.Length = a.Length.length()
	}
	if a.KeywordRules != nil {
		built.KeywordRules = a.KeywordRules.rules()
	}
	if a.MetaRules != nil {
		built.MetaRules = a.MetaRules.rules()
	}
	if a.Images != nil {
		built.Images = a.Images.images()
	}
	if a.Product != nil {
		built.Product = a.Product.product()
	}
	return built
}

func profilesOf(listed []modelProfileArgs) map[domainllm.Role]domainllm.ModelRef {
	if len(listed) == 0 {
		return nil
	}

	profiles := make(map[domainllm.Role]domainllm.ModelRef, len(listed))
	for _, profile := range listed {
		profiles[domainllm.Role(profile.Role)] = domainllm.ModelRef{
			Provider: profile.Provider, Model: profile.Model,
		}
	}
	return profiles
}

func recipeOfSteps(listed []stepArgs) []template.StepSpec {
	if len(listed) == 0 {
		return nil
	}

	recipe := make([]template.StepSpec, 0, len(listed))
	for _, step := range listed {
		recipe = append(recipe, template.StepSpec{Name: step.Name, Enabled: step.Enabled, Params: paramsOf(step)})
	}
	return recipe
}

func paramsOf(step stepArgs) map[string]any {
	params := map[string]any{}
	if step.AllowErrors != nil {
		params["allowErrors"] = *step.AllowErrors
	}
	if step.Iterations != nil {
		params["iterations"] = *step.Iterations
	}
	if len(params) == 0 {
		return nil
	}
	return params
}

func (a templateSpecArgs) patch() (json.RawMessage, error) {
	written := map[string]any{}
	if len(a.Sections) > 0 {
		written["sections"] = sectionsOf(a.Sections)
	}
	if a.Tone != nil {
		written["tone"] = *a.Tone
	}
	if a.Length != nil {
		written["length"] = a.Length.length()
	}
	if a.KeywordRules != nil {
		written["keywordRules"] = a.KeywordRules.rules()
	}
	if a.MetaRules != nil {
		written["metaRules"] = a.MetaRules.rules()
	}
	if a.Images != nil {
		written["images"] = a.Images.images()
	}
	if a.Product != nil {
		written["product"] = a.Product.product()
	}
	if len(a.ModelProfiles) > 0 {
		written["modelProfiles"] = profilesOf(a.ModelProfiles)
	}
	if len(a.Recipe) > 0 {
		written["recipe"] = recipeOfSteps(a.Recipe)
	}
	if len(written) == 0 {
		return nil, errors.New(errors.Invalid, "an override must change at least one field of the specification").
			WithDetail("field", "patch")
	}

	encoded, err := json.Marshal(written)
	if err != nil {
		return nil, errors.Wrap(err, errors.Internal, "encode the template override")
	}
	return encoded, nil
}
