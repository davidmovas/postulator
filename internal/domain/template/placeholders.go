package template

import (
	"regexp"
	"slices"
	"strings"
)

const (
	PlaceholderPrimaryKeyword = "{primaryKeyword}"
	PlaceholderEntityName     = "{entityName}"
	PlaceholderSiteName       = "{siteName}"
	PlaceholderPageTitle      = "{pageTitle}"
)

var placeholders = []string{
	PlaceholderPrimaryKeyword, PlaceholderEntityName, PlaceholderSiteName, PlaceholderPageTitle,
}

var placeholderPattern = regexp.MustCompile(`\{\{[^{}]*\}\}|\{[^{}]*\}`)

func Placeholders() []string {
	return slices.Clone(placeholders)
}

type Vars struct {
	PrimaryKeyword string
	EntityName     string
	SiteName       string
	PageTitle      string
}

func (v Vars) settled() Vars {
	first := func(values ...string) string {
		for _, value := range values {
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				return trimmed
			}
		}
		return ""
	}
	return Vars{
		PrimaryKeyword: first(v.PrimaryKeyword, v.EntityName, v.PageTitle),
		EntityName:     first(v.EntityName, v.PageTitle, v.PrimaryKeyword),
		SiteName:       first(v.SiteName),
		PageTitle:      first(v.PageTitle, v.EntityName, v.PrimaryKeyword),
	}
}

func Expand(text string, vars Vars) string {
	if !strings.Contains(text, "{") {
		return text
	}
	values := vars.settled()
	replaced := strings.NewReplacer(
		PlaceholderPrimaryKeyword, values.PrimaryKeyword,
		PlaceholderEntityName, values.EntityName,
		PlaceholderSiteName, values.SiteName,
		PlaceholderPageTitle, values.PageTitle,
	).Replace(text)
	return strings.Join(strings.Fields(replaced), " ")
}

func UnknownPlaceholders(text string) []string {
	unknown := make([]string, 0)
	for _, token := range placeholderPattern.FindAllString(text, -1) {
		if slices.Contains(placeholders, token) || slices.Contains(unknown, token) {
			continue
		}
		unknown = append(unknown, token)
	}
	return unknown
}

func (s TemplateSpec) Expanded(vars Vars) TemplateSpec {
	expanded := s
	expanded.Sections = make([]Section, len(s.Sections))
	for i := range s.Sections {
		section := s.Sections[i]
		section.Heading = Expand(section.Heading, vars)
		section.Intent = Expand(section.Intent, vars)
		if len(section.KeywordRules.Include) > 0 {
			section.KeywordRules.Include = make([]string, len(s.Sections[i].KeywordRules.Include))
			for j, phrase := range s.Sections[i].KeywordRules.Include {
				section.KeywordRules.Include[j] = Expand(phrase, vars)
			}
		}
		expanded.Sections[i] = section
	}
	expanded.MetaRules.TitlePattern = Expand(s.MetaRules.TitlePattern, vars)
	return expanded
}
