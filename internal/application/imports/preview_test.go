package imports_test

import (
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestInspectOpensTheFilePreMapped(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	path := h.file(t, "client.csv", "URL;Page Title;Secondary Keywords\n/a/;A;one\n/b/;B;two\n/c/;C;three\n/d/;D;four\n/e/;E;five\n/f/;F;six\n")

	got, err := h.service.Inspect(t.Context(), imports.InspectRequest{SiteID: h.siteID, Path: path})
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if len(got.Headers) != 3 || got.Rows != 6 || len(got.Sample) != 5 {
		t.Fatalf("headers = %v, rows = %d, sample = %d", got.Headers, got.Rows, len(got.Sample))
	}
	if got.Detected.Columns[string(importmap.FieldPath)] != "URL" ||
		got.Detected.Columns[string(importmap.FieldTitle)] != "Page Title" ||
		got.Detected.Columns[string(importmap.FieldKeywords)] != "Secondary Keywords" {
		t.Fatalf("detected = %v", got.Detected.Columns)
	}
	if got.Detected.SiteID != h.siteID || len(got.Saved) != 0 {
		t.Fatalf("detected site = %s, saved = %v", got.Detected.SiteID, got.Saved)
	}
}

func TestInspectRefusesWhatItCannotOpen(t *testing.T) {
	t.Parallel()

	h := newHarnessWithRows(t, 100)

	cases := []struct {
		name string
		req  func(*testing.T) imports.InspectRequest
		code errors.Code
	}{
		{
			name: "no site",
			req: func(t *testing.T) imports.InspectRequest {
				return imports.InspectRequest{Path: h.file(t, "a.csv", "path\n/\n")}
			},
			code: errors.Invalid,
		},
		{
			name: "unknown site",
			req: func(t *testing.T) imports.InspectRequest {
				return imports.InspectRequest{SiteID: "00000000-0000-4000-8000-000000000000", Path: h.file(t, "b.csv", "path\n/\n")}
			},
			code: errors.NotFound,
		},
		{
			name: "not a sheet",
			req: func(t *testing.T) imports.InspectRequest {
				return imports.InspectRequest{SiteID: h.siteID, Path: h.file(t, "c.txt", "path\n/\n")}
			},
			code: errors.Invalid,
		},
		{
			name: "above the row ceiling",
			req: func(t *testing.T) imports.InspectRequest {
				body := strings.Builder{}
				body.WriteString("path\n")
				for i := range 101 {
					body.WriteString("/p" + string(rune('a'+i%26)) + string(rune('a'+i/26)) + "/\n")
				}
				return imports.InspectRequest{SiteID: h.siteID, Path: h.file(t, "d.csv", body.String())}
			},
			code: errors.Invalid,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := h.service.Inspect(t.Context(), tc.req(t)); !errors.IsCode(err, tc.code) {
				t.Fatalf("Inspect = %v, want %s", err, tc.code)
			}
		})
	}
}

func TestPreviewReadsTheGraphWithoutWriting(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	report := h.preview(t, h.file(t, "graph.csv", graphSheet), graphMapping(h))

	if len(report.Errors) != 0 {
		t.Fatalf("errors = %+v", report.Errors)
	}
	if len(report.Entities) != 3 || len(report.Pages) != 3 {
		t.Fatalf("entities = %d, pages = %d", len(report.Entities), len(report.Pages))
	}

	hosting, found := entity(report, "Hosting")
	if !found || hosting.Action != string(imports.ActionCreate) || hosting.Kind != "topic" {
		t.Fatalf("hosting = %+v", hosting)
	}
	if len(hosting.Keywords) != 2 || hosting.Keywords[0] != "hosting" || hosting.Keywords[1] != "servers" {
		t.Fatalf("hosting keywords = %v", hosting.Keywords)
	}
	if len(report.Edges) != 3 {
		t.Fatalf("edges = %+v", report.Edges)
	}
	if len(h.entities(t)) != 0 || len(h.pages(t)) != 0 {
		t.Fatal("the preview wrote to the database")
	}
}

func TestPreviewFlagsWhatTheSheetGetsWrong(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		sheet    string
		code     imports.FindingCode
		errored  bool
		expected int
	}{
		{
			name:     "an unknown parent",
			sheet:    "path,entity,parent\n/a/,Alpha,Absent\n",
			code:     imports.CodeUnknownParent,
			errored:  true,
			expected: 1,
		},
		{
			name:     "an unknown related entity",
			sheet:    "path,entity,related\n/a/,Alpha,Absent\n",
			code:     imports.CodeUnknownRelated,
			errored:  true,
			expected: 1,
		},
		{
			name:     "an entity that parents itself",
			sheet:    "path,entity,parent\n/a/,Alpha,alpha\n",
			code:     imports.CodeSelfEdge,
			errored:  true,
			expected: 1,
		},
		{
			name:     "a cycle between two entities",
			sheet:    "path,entity,parent\n/a/,Alpha,Beta\n/b/,Beta,Alpha\n",
			code:     imports.CodeCycle,
			errored:  true,
			expected: 1,
		},
		{
			name:     "a path that cannot be read",
			sheet:    "path,entity,parent\n/a/../b/,Alpha,\n",
			code:     imports.CodeBadPath,
			errored:  true,
			expected: 1,
		},
		{
			name:     "a repeated path",
			sheet:    "path,entity,parent\n/a/,Alpha,\n/A/,Alpha,\n",
			code:     imports.CodeDuplicatePath,
			errored:  false,
			expected: 1,
		},
		{
			name:     "a missing intermediate path",
			sheet:    "path,entity,parent\n/a/b/c/,Alpha,\n",
			code:     imports.CodeIntermediatePath,
			errored:  false,
			expected: 3,
		},
		{
			name:     "a row that names nothing",
			sheet:    "path,entity,parent\n,,Alpha\n",
			code:     imports.CodeNoTarget,
			errored:  false,
			expected: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			mapping := h.mapping(map[string]string{
				string(importmap.FieldPath):         "path",
				string(importmap.FieldEntity):       "entity",
				string(importmap.FieldParentEntity): "parent",
			})
			if strings.Contains(tc.sheet, "related") {
				mapping.Columns = map[string]string{
					string(importmap.FieldPath):    "path",
					string(importmap.FieldEntity):  "entity",
					string(importmap.FieldRelated): "related",
				}
			}

			report := h.preview(t, h.file(t, "sheet.csv", tc.sheet), mapping)
			list := report.Warnings
			if tc.errored {
				list = report.Errors
			}
			if got := findings(list, tc.code); len(got) != tc.expected {
				t.Fatalf("%s findings = %+v, want %d; report = %+v", tc.code, got, tc.expected, report)
			}
		})
	}
}

func TestPreviewCreatesTheIntermediatePathsItNeeds(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	report := h.preview(t, h.file(t, "deep.csv", "path,title\n/services/web/react/,React\n"),
		h.mapping(map[string]string{string(importmap.FieldPath): "path", string(importmap.FieldTitle): "title"}))

	for _, path := range []string{"/", "/services/", "/services/web/", "/services/web/react/"} {
		found, ok := page(report, path)
		if !ok {
			t.Fatalf("%s is missing from %+v", path, report.Pages)
		}
		if path == "/services/web/react/" {
			if found.Generated || found.Title != "React" {
				t.Fatalf("the named page = %+v", found)
			}
			continue
		}
		if !found.Generated {
			t.Fatalf("%s is not flagged as generated", path)
		}
	}
	if title, _ := page(report, "/services/web/"); title.Title != "Web" {
		t.Fatalf("the generated title = %q, want Web", title.Title)
	}
	if home, _ := page(report, "/"); home.Title != "Home" {
		t.Fatalf("the root title = %q, want Home", home.Title)
	}
}

func TestPreviewReportsCannibalizationAgainstTheSite(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	first := h.file(t, "first.csv", "path,entity,primary keyword\n/hosting/,Hosting,cheap hosting\n")
	mapping := h.mapping(map[string]string{
		string(importmap.FieldPath):           "path",
		string(importmap.FieldEntity):         "entity",
		string(importmap.FieldPrimaryKeyword): "primary keyword",
	})
	h.apply(t, first, mapping)

	second := h.file(t, "second.csv", "path,entity,primary keyword\n/servers/,Servers,cheap hosting\n")
	report := h.preview(t, second, mapping)

	if len(report.Cannibalization) != 1 {
		t.Fatalf("cannibalization = %+v", report.Cannibalization)
	}
	if report.Cannibalization[0].Path != "/hosting/" || report.Cannibalization[0].Reason != "same_primary_keyword" {
		t.Fatalf("evidence = %+v", report.Cannibalization[0])
	}
	if len(findings(report.Warnings, imports.CodeCannibalization)) != 1 {
		t.Fatalf("warnings = %+v", report.Warnings)
	}
	if len(report.Errors) != 0 {
		t.Fatalf("cannibalization must not block the import: %+v", report.Errors)
	}
}

func TestPreviewFallsBackOnUnknownVocabulary(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	report := h.preview(t, h.file(t, "kinds.csv", "path,entity,entity type,post type,page type\n/a/,Alpha,widget,gadget,sprocket\n"),
		h.mapping(map[string]string{
			string(importmap.FieldPath):       "path",
			string(importmap.FieldEntity):     "entity",
			string(importmap.FieldEntityKind): "entity type",
			string(importmap.FieldWPType):     "post type",
			string(importmap.FieldPageKind):   "page type",
		}))

	for _, code := range []imports.FindingCode{imports.CodeUnknownEntityKind, imports.CodeUnknownWPType, imports.CodeUnknownPageKind} {
		if len(findings(report.Warnings, code)) != 1 {
			t.Fatalf("%s findings = %+v", code, report.Warnings)
		}
	}
	if alpha, _ := entity(report, "Alpha"); alpha.Kind != "topic" {
		t.Fatalf("kind = %q, want topic", alpha.Kind)
	}
	if found, _ := page(report, "/a/"); found.WPType != "page" {
		t.Fatalf("wordpress type = %q, want page", found.WPType)
	}
}

func TestPreviewRefusesAMappingTheFileDoesNotCarry(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	_, err := h.service.Preview(t.Context(), imports.PreviewRequest{
		SiteID:  h.siteID,
		Path:    h.file(t, "thin.csv", "path\n/a/\n"),
		Mapping: h.mapping(map[string]string{string(importmap.FieldPath): "path", string(importmap.FieldTitle): "title"}),
	})
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Preview = %v, want an invalid error", err)
	}
}

func TestTheFindingCodesDeclareWhichOnesBlockAnApply(t *testing.T) {
	t.Parallel()

	cases := []struct {
		code     imports.FindingCode
		blocking bool
	}{
		{code: imports.CodeBadPath, blocking: true},
		{code: imports.CodeNoTarget, blocking: false},
		{code: imports.CodeDuplicatePath, blocking: false},
		{code: imports.CodeIntermediatePath, blocking: false},
		{code: imports.CodeUnknownParent, blocking: true},
		{code: imports.CodeUnknownRelated, blocking: true},
		{code: imports.CodeSelfEdge, blocking: true},
		{code: imports.CodeCycle, blocking: true},
		{code: imports.CodeCannibalization, blocking: false},
		{code: imports.CodeUnknownEntityKind, blocking: false},
		{code: imports.CodeUnknownPageKind, blocking: false},
		{code: imports.CodeUnknownWPType, blocking: false},
		{code: imports.FindingCode("invented"), blocking: false},
	}

	for _, tc := range cases {
		t.Run(string(tc.code), func(t *testing.T) {
			t.Parallel()

			if got := tc.code.Blocking(); got != tc.blocking {
				t.Fatalf("%s.Blocking() = %v, want %v", tc.code, got, tc.blocking)
			}
		})
	}
}
