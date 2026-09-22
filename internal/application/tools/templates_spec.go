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
	Enabled     bool   `json:"enabled" description:"Run this step; a disabled step stays in the recipe and is skipped"`
	AllowErrors *bool  `json:"allowErrors,omitempty" description:"Only the validate step reads this: let the item go on although validation found faults"`
	Iterations  *int   `json:"iterations,omitempty" minimum:"1" description:"Only the repair_links step reads this: how many passes it may make"`
}

type templateSpecArgs struct {
	Sections      []template.Section    `json:"sections" description:"The sections the page is built from, in the order they appear; at least one is required"`
	Tone          string                `json:"tone" description:"How the page should read, one or two sentences to the writer"`
	Length        template.Length       `json:"length" description:"How long the whole page may run to"`
	KeywordRules  template.KeywordRules `json:"keywordRules" description:"Where the primary keyword has to appear and how often it may"`
	LinkRules     template.LinkRules    `json:"linkRules" description:"How many internal links the page carries and which of them it owes"`
	MetaRules     template.MetaRules    `json:"metaRules" description:"How the SEO title and description are built"`
	Images        template.Images       `json:"images" description:"Whether the page carries images and where they come from"`
	ModelProfiles []modelProfileArgs    `json:"modelProfiles,omitempty" description:"Which model does which job for this template; leave it out to take the site profiles"`
	Recipe        []stepArgs            `json:"recipe,omitempty" description:"The steps a run takes for this template, in order; leave it out for the default recipe"`
}

func (a templateSpecArgs) spec() template.TemplateSpec {
	return template.TemplateSpec{
		Sections:      a.Sections,
		Tone:          a.Tone,
		Length:        a.Length,
		KeywordRules:  a.KeywordRules,
		LinkRules:     a.LinkRules,
		MetaRules:     a.MetaRules,
		Images:        a.Images,
		ModelProfiles: profilesOf(a.ModelProfiles),
		Recipe:        recipeOfSteps(a.Recipe),
	}
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
	Sections      *[]template.Section    `json:"sections,omitempty" description:"The whole new list of sections, left out to keep the current one"`
	Tone          *string                `json:"tone,omitempty" description:"The new tone, left out to keep the current one"`
	Length        *template.Length       `json:"length,omitempty" description:"The new length window, left out to keep the current one"`
	KeywordRules  *template.KeywordRules `json:"keywordRules,omitempty" description:"The new keyword rules, left out to keep the current ones"`
	LinkRules     *template.LinkRules    `json:"linkRules,omitempty" description:"The new link rules, left out to keep the current ones"`
	MetaRules     *template.MetaRules    `json:"metaRules,omitempty" description:"The new meta rules, left out to keep the current ones"`
	Images        *template.Images       `json:"images,omitempty" description:"The new image settings, left out to keep the current ones"`
	ModelProfiles []modelProfileArgs     `json:"modelProfiles,omitempty" description:"The whole new set of model profiles, left out to keep the current ones"`
	Recipe        []stepArgs             `json:"recipe,omitempty" description:"The whole new recipe in order, left out to keep the current one"`
}

func (a templatePatchArgs) patch() (json.RawMessage, error) {
	written := map[string]any{}
	if a.Sections != nil {
		written["sections"] = *a.Sections
	}
	if a.Tone != nil {
		written["tone"] = *a.Tone
	}
	if a.Length != nil {
		written["length"] = *a.Length
	}
	if a.KeywordRules != nil {
		written["keywordRules"] = *a.KeywordRules
	}
	if a.LinkRules != nil {
		written["linkRules"] = *a.LinkRules
	}
	if a.MetaRules != nil {
		written["metaRules"] = *a.MetaRules
	}
	if a.Images != nil {
		written["images"] = *a.Images
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
