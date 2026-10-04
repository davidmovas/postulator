package template_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/template"
)

func TestExpandFillsTheKnownPlaceholders(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		text string
		vars template.Vars
		want string
	}{
		{
			name: "every placeholder is filled from its own value",
			text: "{primaryKeyword} by {entityName} on {siteName}: {pageTitle}",
			vars: template.Vars{PrimaryKeyword: "running shoes", EntityName: "Running Shoes", SiteName: "Shop", PageTitle: "Best pairs"},
			want: "running shoes by Running Shoes on Shop: Best pairs",
		},
		{
			name: "a missing keyword falls back to the entity name",
			text: "Best {primaryKeyword} for beginners",
			vars: template.Vars{EntityName: "Trail Shoes"},
			want: "Best Trail Shoes for beginners",
		},
		{
			name: "a missing entity falls back to the page title",
			text: "About {entityName}",
			vars: template.Vars{PageTitle: "Hiking boots"},
			want: "About Hiking boots",
		},
		{
			name: "a missing page title falls back to the entity name",
			text: "{pageTitle} | {siteName}",
			vars: template.Vars{EntityName: "Boots", SiteName: "Shop"},
			want: "Boots | Shop",
		},
		{
			name: "a value nobody knows leaves no double space behind",
			text: "How to {primaryKeyword} today",
			vars: template.Vars{},
			want: "How to today",
		},
		{
			name: "text without placeholders is untouched",
			text: "Frequently Asked Questions",
			vars: template.Vars{PrimaryKeyword: "x"},
			want: "Frequently Asked Questions",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := template.Expand(tc.text, tc.vars); got != tc.want {
				t.Fatalf("Expand(%q) = %q, want %q", tc.text, got, tc.want)
			}
		})
	}
}

func TestUnknownPlaceholdersNamesWhatCannotBeFilled(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		text string
		want []string
	}{
		{name: "known ones pass", text: "{primaryKeyword} | {siteName} {entityName} {pageTitle}", want: []string{}},
		{name: "a name nobody declared", text: "Best {entity} for {primaryKeyword}", want: []string{"{entity}"}},
		{name: "double braces are not a placeholder", text: "Best {{primaryKeyword}} shoes", want: []string{"{{primaryKeyword}}"}},
		{name: "each unknown once", text: "{x} and {x} and {y}", want: []string{"{x}", "{y}"}},
		{name: "plain text", text: "Overview", want: []string{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := template.UnknownPlaceholders(tc.text); !slices.Equal(got, tc.want) {
				t.Fatalf("UnknownPlaceholders(%q) = %v, want %v", tc.text, got, tc.want)
			}
		})
	}
}

func TestPlaceholdersAreTheClosedList(t *testing.T) {
	t.Parallel()

	want := []string{"{primaryKeyword}", "{entityName}", "{siteName}", "{pageTitle}"}
	if got := template.Placeholders(); !slices.Equal(got, want) {
		t.Fatalf("Placeholders() = %v, want %v", got, want)
	}
}

func TestExpandedSpecFillsHeadingsIntentsPinnedPhrasesAndTheTitlePattern(t *testing.T) {
	t.Parallel()

	spec := validSpec()
	spec.Sections[0].Heading = "Why {primaryKeyword} matter"
	spec.Sections[0].Intent = "Say what {entityName} is"
	spec.Sections[0].KeywordRules.Include = []string{"{primaryKeyword} guide"}
	spec.MetaRules.TitlePattern = "{primaryKeyword} | {siteName}"

	expanded := spec.Expanded(template.Vars{PrimaryKeyword: "trail shoes", EntityName: "Trail Shoes", SiteName: "Shop"})

	if expanded.Sections[0].Heading != "Why trail shoes matter" {
		t.Fatalf("heading = %q", expanded.Sections[0].Heading)
	}
	if expanded.Sections[0].Intent != "Say what Trail Shoes is" {
		t.Fatalf("intent = %q", expanded.Sections[0].Intent)
	}
	if expanded.Sections[0].KeywordRules.Include[0] != "trail shoes guide" {
		t.Fatalf("include = %v", expanded.Sections[0].KeywordRules.Include)
	}
	if expanded.MetaRules.TitlePattern != "trail shoes | Shop" {
		t.Fatalf("title pattern = %q", expanded.MetaRules.TitlePattern)
	}
	if spec.Sections[0].Heading != "Why {primaryKeyword} matter" || spec.Sections[0].KeywordRules.Include[0] != "{primaryKeyword} guide" {
		t.Fatalf("the source spec was changed in place: %+v", spec.Sections[0])
	}
}

func TestExpandedSpecFillsTheProductOutputsOnACopy(t *testing.T) {
	t.Parallel()

	spec := validSpec()
	spec.Product = &template.Product{
		ShortDescription: template.ProductShortDescription{Enabled: true, Intent: "Say why {entityName} sells"},
		Specifications: []template.ProductSpecification{
			{Name: "Form", Intent: "The form {entityName} comes in"},
		},
	}

	expanded := spec.Expanded(template.Vars{EntityName: "BPC-157 Liquid"})

	if expanded.Product == spec.Product {
		t.Fatal("the expanded spec shares the product block with its source")
	}
	if expanded.Product.ShortDescription.Intent != "Say why BPC-157 Liquid sells" ||
		expanded.Product.Specifications[0].Intent != "The form BPC-157 Liquid comes in" {
		t.Fatalf("product = %+v", expanded.Product)
	}
	if spec.Product.ShortDescription.Intent != "Say why {entityName} sells" ||
		spec.Product.Specifications[0].Intent != "The form {entityName} comes in" {
		t.Fatalf("the source product block was changed in place: %+v", spec.Product)
	}

	if bare := validSpec().Expanded(template.Vars{}); bare.Product != nil {
		t.Errorf("a spec without a product block gained one: %+v", bare.Product)
	}
}

func TestASpecWithoutAProductBlockEncodesAsBefore(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(validSpec())
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(string(encoded), `"product"`) {
		t.Errorf("a spec without products encodes a product key: %s", encoded)
	}
}
