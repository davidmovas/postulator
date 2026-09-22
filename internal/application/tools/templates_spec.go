package tools

import (
	"encoding/json"

	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type modelProfileArgs struct {
	Role     string `json:"role" enum:"writer,editor,linker,judge,chat,image,titler" description:"Which job this model takes while the page is written"`
	Provider string `json:"provider" description:"The provider to use for that job, such as openai"`
	Model    string `json:"model" description:"The model to use for that job"`
}

type stepArgs struct {
	Name        string `json:"name" enum:"resolve_context,generate_body,generate_meta,insert_links,repair_links,generate_images,validate,judge,repair_hierarchy,publish,relink_neighbors,sync_back,report,sync_site" description:"Which step of the run this is"`
	Enabled     bool   `json:"enabled,omitempty" description:"Run this step; leave it out and the step stays in the recipe and is skipped"`
	AllowErrors *bool  `json:"allowErrors,omitempty" description:"Only the validate step reads this: let the item go on although validation found faults"`
	Iterations  *int   `json:"iterations,omitempty" minimum:"1" description:"Only the repair_links step reads this: how many passes it may make"`
}

type sectionKeywordRulesArgs struct {
	Include          []string `json:"include,omitempty" description:"Phrases this section must use at least once; leave it out for none"`
	PrimaryInHeading bool     `json:"primaryInHeading,omitempty" description:"The primary keyword must appear in this section's heading; leave it out and it need not"`
}

func (a sectionKeywordRulesArgs) rules() template.SectionKeywordRules {
	return template.SectionKeywordRules{Include: a.Include, PrimaryInHeading: a.PrimaryInHeading}
}

type sectionArgs struct {
	Heading      string                   `json:"heading" description:"The heading the section opens with, which may carry {primaryKeyword}"`
	Intent       string                   `json:"intent,omitempty" description:"What the section has to cover, one short sentence to the writer"`
	TargetWords  int                      `json:"targetWords,omitempty" minimum:"0" description:"About how many words the section should run to; leave it out to let the writer decide"`
	Required     bool                     `json:"required,omitempty" description:"The page is not valid without this section; leave it out to make it optional"`
	KeywordRules *sectionKeywordRulesArgs `json:"keywordRules,omitempty" description:"What this section has to say about the keywords; leave it out to ask nothing of it"`
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
	Min int `json:"min,omitempty" minimum:"0" description:"The fewest words the whole page may run to; leave it out for no floor"`
	Max int `json:"max,omitempty" minimum:"0" description:"The most words the whole page may run to; leave it out for no ceiling"`
}

func (a lengthArgs) length() template.Length {
	return template.Length{Min: a.Min, Max: a.Max}
}

type keywordRulesArgs struct {
	PrimaryInTitle          bool    `json:"primaryInTitle,omitempty" description:"The primary keyword must appear in the title; leave it out and it need not"`
	PrimaryInH1             bool    `json:"primaryInH1,omitempty" description:"The primary keyword must appear in the first heading; leave it out and it need not"`
	PrimaryInFirstParagraph bool    `json:"primaryInFirstParagraph,omitempty" description:"The primary keyword must appear in the opening paragraph; leave it out and it need not"`
	MaxDensity              float64 `json:"maxDensity,omitempty" minimum:"0" maximum:"1" description:"The largest share of the words the primary keyword may take, between 0 and 1; leave it out for no ceiling"`
}

func (a keywordRulesArgs) rules() template.KeywordRules {
	return template.KeywordRules{
		PrimaryInTitle:          a.PrimaryInTitle,
		PrimaryInH1:             a.PrimaryInH1,
		PrimaryInFirstParagraph: a.PrimaryInFirstParagraph,
		MaxDensity:              a.MaxDensity,
	}
}

type linkRulesArgs struct {
	UpDepth                    int     `json:"upDepth,omitempty" minimum:"0" description:"How many levels up the tree a page links to, 1 for its parent alone; leave it out for none"`
	DownLinks                  bool    `json:"downLinks,omitempty" description:"Link down to the children of the entity; leave it out and none are placed"`
	SiblingMinWeight           float64 `json:"siblingMinWeight,omitempty" minimum:"0" maximum:"1" description:"The weight a related edge needs before a sibling link is placed, between 0 and 1; leave it out to place every sibling link"`
	MaxLinks                   int     `json:"maxLinks,omitempty" minimum:"0" description:"The most internal links one page may carry; leave it out for no ceiling"`
	MaxPerTarget               int     `json:"maxPerTarget,omitempty" minimum:"0" description:"The most links one page may point at a single target; leave it out for no ceiling"`
	ParentLinkWithinParagraphs int     `json:"parentLinkWithinParagraphs,omitempty" minimum:"0" description:"The parent link must appear within this many paragraphs of the start; leave it out to let it sit anywhere"`
	ChildrenSection            bool    `json:"childrenSection,omitempty" description:"Close the page with a section listing its children; leave it out for no such section"`
}

func (a linkRulesArgs) rules() template.LinkRules {
	return template.LinkRules{
		UpDepth:                    a.UpDepth,
		DownLinks:                  a.DownLinks,
		SiblingMinWeight:           a.SiblingMinWeight,
		MaxLinks:                   a.MaxLinks,
		MaxPerTarget:               a.MaxPerTarget,
		ParentLinkWithinParagraphs: a.ParentLinkWithinParagraphs,
		ChildrenSection:            a.ChildrenSection,
	}
}

type metaRulesArgs struct {
	TitlePattern   string `json:"titlePattern,omitempty" description:"How to build the SEO title, for example {primaryKeyword} | {siteName}; leave it out to use the page title"`
	DescriptionMax int    `json:"descriptionMax,omitempty" minimum:"0" description:"The most characters the SEO description may run to; leave it out for no ceiling"`
}

func (a metaRulesArgs) rules() template.MetaRules {
	return template.MetaRules{TitlePattern: a.TitlePattern, DescriptionMax: a.DescriptionMax}
}

type imagesArgs struct {
	Featured bool                 `json:"featured,omitempty" description:"The page carries a featured image; leave it out for a page without one"`
	Inline   int                  `json:"inline,omitempty" minimum:"0" description:"How many images to place inside the body; leave it out for a page without images"`
	Source   template.ImageSource `json:"source,omitempty" enum:"ai,wpmedia,local" description:"Where the images come from: drawn by a model, picked from the WordPress library, or read from a folder; required as soon as any image is asked for"`
}

func (a imagesArgs) images() template.Images {
	return template.Images{Featured: a.Featured, Inline: a.Inline, Source: a.Source}
}

type templateSpecArgs struct {
	Sections      []sectionArgs      `json:"sections" description:"The sections the page is built from, in the order they appear; at least one is required"`
	Tone          string             `json:"tone,omitempty" description:"How the page should read, one or two sentences to the writer"`
	Length        *lengthArgs        `json:"length,omitempty" description:"How long the whole page may run to; leave it out for no length window"`
	KeywordRules  *keywordRulesArgs  `json:"keywordRules,omitempty" description:"Where the primary keyword has to appear and how often it may; leave it out to ask nothing"`
	LinkRules     *linkRulesArgs     `json:"linkRules,omitempty" description:"How many internal links the page carries and which of them it owes; leave it out to ask nothing"`
	MetaRules     *metaRulesArgs     `json:"metaRules,omitempty" description:"How the SEO title and description are built; leave it out to take the page title"`
	Images        *imagesArgs        `json:"images,omitempty" description:"Whether the page carries images and where they come from; leave it out for a page without images"`
	ModelProfiles []modelProfileArgs `json:"modelProfiles,omitempty" description:"Which model does which job for this template; leave it out to take the site profiles"`
	Recipe        []stepArgs         `json:"recipe,omitempty" description:"The steps a run takes for this template, in order; leave it out for the default recipe"`
}

func (a templateSpecArgs) spec() template.TemplateSpec {
	built := template.TemplateSpec{
		Sections:      sectionsOf(a.Sections),
		Tone:          a.Tone,
		ModelProfiles: profilesOf(a.ModelProfiles),
		Recipe:        recipeOfSteps(a.Recipe),
	}
	if a.Length != nil {
		built.Length = a.Length.length()
	}
	if a.KeywordRules != nil {
		built.KeywordRules = a.KeywordRules.rules()
	}
	if a.LinkRules != nil {
		built.LinkRules = a.LinkRules.rules()
	}
	if a.MetaRules != nil {
		built.MetaRules = a.MetaRules.rules()
	}
	if a.Images != nil {
		built.Images = a.Images.images()
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

type templatePatchArgs struct {
	Sections      *[]sectionArgs     `json:"sections,omitempty" description:"The whole new list of sections, left out to keep the current one"`
	Tone          *string            `json:"tone,omitempty" description:"The new tone, left out to keep the current one"`
	Length        *lengthArgs        `json:"length,omitempty" description:"The new length window, left out to keep the current one"`
	KeywordRules  *keywordRulesArgs  `json:"keywordRules,omitempty" description:"The new keyword rules, left out to keep the current ones"`
	LinkRules     *linkRulesArgs     `json:"linkRules,omitempty" description:"The new link rules, left out to keep the current ones"`
	MetaRules     *metaRulesArgs     `json:"metaRules,omitempty" description:"The new meta rules, left out to keep the current ones"`
	Images        *imagesArgs        `json:"images,omitempty" description:"The new image settings, left out to keep the current ones"`
	ModelProfiles []modelProfileArgs `json:"modelProfiles,omitempty" description:"The whole new set of model profiles, left out to keep the current ones"`
	Recipe        []stepArgs         `json:"recipe,omitempty" description:"The whole new recipe in order, left out to keep the current one"`
}

func (a templatePatchArgs) patch() (json.RawMessage, error) {
	written := map[string]any{}
	if a.Sections != nil {
		written["sections"] = sectionsOf(*a.Sections)
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
	if a.LinkRules != nil {
		written["linkRules"] = a.LinkRules.rules()
	}
	if a.MetaRules != nil {
		written["metaRules"] = a.MetaRules.rules()
	}
	if a.Images != nil {
		written["images"] = a.Images.images()
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
