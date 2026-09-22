package steps_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const longDescription = "Espresso is a way to make coffee under pressure and this description is deliberately " +
	"far longer than the hundred and fifty five characters a search engine will ever show a reader of it."

func metaContext(t *testing.T, reply string) (*run.StepContext, steps.Deps) {
	t.Helper()

	deps := unitDeps()
	deps.LLM = llmStub{reply: reply}
	sc := unitContext(t, map[run.ArtifactKind][]byte{run.ArtifactDraft: []byte(goodDraft)})
	sc.Spec.MetaRules = template.MetaRules{TitlePattern: "{primaryKeyword} | {siteName}", DescriptionMax: 155}
	return sc, deps
}

func runMeta(t *testing.T, reply string) steps.Meta {
	t.Helper()

	sc, deps := metaContext(t, reply)
	result, err := steps.GenerateMeta(deps).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("GenerateMeta: %v", err)
	}
	if len(result.Artifacts) != 1 || result.Artifacts[0].Kind != run.ArtifactMeta {
		t.Fatalf("GenerateMeta produced %+v", result.Artifacts)
	}

	var meta steps.Meta
	if err = json.Unmarshal(result.Artifacts[0].Blob, &meta); err != nil {
		t.Fatalf("decode the meta artifact: %v", err)
	}
	return meta
}

func TestGenerateMetaSettlesWhatTheModelReturns(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		reply string
		check func(*testing.T, steps.Meta)
	}{
		{
			name:  "a complete answer is kept",
			reply: `{"title":"espresso | Shop","description":"How to pull a shot.","canonical":"https://shop.example.com/coffee/espresso/","ogTitle":"Espresso","ogDescription":"Pull a better shot."}`,
			check: func(t *testing.T, meta steps.Meta) {
				if meta.Title != "espresso | Shop" || meta.OGTitle != "Espresso" {
					t.Errorf("meta = %+v", meta)
				}
			},
		},
		{
			name:  "empty fields fall back to the draft and the page",
			reply: `{"title":"","description":"","canonical":"","ogTitle":"","ogDescription":""}`,
			check: func(t *testing.T, meta steps.Meta) {
				if meta.Title != "Espresso guide" {
					t.Errorf("title = %q, want the draft title", meta.Title)
				}
				if len(meta.Findings) != 1 || meta.Findings[0].Code != steps.CodeMetaNotWritten {
					t.Errorf("findings = %+v, want the borrowed snippet named", meta.Findings)
				}
				if meta.Description != "A short guide to espresso." {
					t.Errorf("description = %q, want the draft summary", meta.Description)
				}
				if meta.Canonical != "https://shop.example.com/coffee/espresso/" {
					t.Errorf("canonical = %q", meta.Canonical)
				}
				if meta.OGTitle != meta.Title || meta.OGDescription != meta.Description {
					t.Errorf("the open graph pair did not fall back: %+v", meta)
				}
			},
		},
		{
			name:  "a long description is clipped to the rule",
			reply: `{"title":"espresso | Shop","description":"` + longDescription + `"}`,
			check: func(t *testing.T, meta steps.Meta) {
				if len([]rune(meta.Description)) > 155 {
					t.Errorf("description is %d characters: %q", len([]rune(meta.Description)), meta.Description)
				}
				if !strings.HasSuffix(meta.Description, "…") {
					t.Errorf("description = %q, want it to end in an ellipsis", meta.Description)
				}
				if meta.OGDescription != meta.Description {
					t.Errorf("the open graph description = %q", meta.OGDescription)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.check(t, runMeta(t, tc.reply))
		})
	}
}

func TestGenerateMetaSaysWhenTheSnippetIsOnlyTheDraftAgain(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		reply string
		want  []string
	}{
		{
			name:  "the model answered with an empty object",
			reply: `{}`,
			want:  []string{"title", "description"},
		},
		{
			name:  "the model wrote a title and nothing else",
			reply: `{"title":"espresso | Shop"}`,
			want:  []string{"description"},
		},
		{
			name:  "the model answered in full",
			reply: `{"title":"espresso | Shop","description":"How to pull a shot."}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			meta := runMeta(t, tc.reply)
			if tc.want == nil {
				if len(meta.Findings) != 0 {
					t.Fatalf("findings = %+v, want none", meta.Findings)
				}
				return
			}

			if len(meta.Findings) != 1 {
				t.Fatalf("findings = %+v, want one", meta.Findings)
			}
			finding := meta.Findings[0]
			if finding.Code != steps.CodeMetaNotWritten || finding.Severity != content.SeverityWarn {
				t.Errorf("finding = %+v", finding)
			}
			if !strings.Contains(finding.Message, "/coffee/espresso/") {
				t.Errorf("message = %q, want the page named", finding.Message)
			}
			if fields := detailList(finding.Details["fields"]); !slices.Equal(fields, tc.want) {
				t.Errorf("fields = %v, want %v", fields, tc.want)
			}
		})
	}
}

func detailList(held any) []string {
	values, ok := held.([]any)
	if !ok {
		return nil
	}

	out := make([]string, 0, len(values))
	for _, value := range values {
		text, isText := value.(string)
		if !isText {
			return nil
		}
		out = append(out, text)
	}
	return out
}

func TestGenerateMetaCanonicalJoinsTheOriginOnce(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		baseURL string
		path    string
		want    string
	}{
		{
			name: "a root install", baseURL: "https://shop.example.com", path: "/coffee/espresso/",
			want: "https://shop.example.com/coffee/espresso/",
		},
		{
			name: "a subdirectory install", baseURL: "https://shop.example.com/blog", path: "/blog/coffee/espresso/",
			want: "https://shop.example.com/blog/coffee/espresso/",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sc, deps := metaContext(t, `{"title":"espresso | Shop","canonical":""}`)
			deps.Sites = siteStub{record: site.Site{
				ID: "site", Name: "Shop", BaseURL: tc.baseURL, Username: "editor", Status: site.StatusActive,
			}}
			sc.Page.Path = tc.path

			result, err := steps.GenerateMeta(deps).Run(t.Context(), sc)
			if err != nil {
				t.Fatalf("GenerateMeta: %v", err)
			}
			var meta steps.Meta
			if err = json.Unmarshal(result.Artifacts[0].Blob, &meta); err != nil {
				t.Fatalf("decode the meta artifact: %v", err)
			}
			if meta.Canonical != tc.want {
				t.Errorf("canonical = %q, want %q", meta.Canonical, tc.want)
			}
		})
	}
}

func TestGenerateMetaCarriesThePatternIntoThePrompt(t *testing.T) {
	t.Parallel()

	sc, deps := metaContext(t, `{"title":"espresso | Shop"}`)
	recorder := &promptRecorder{reply: `{"title":"espresso | Shop"}`}
	deps.LLM = recorder

	if _, err := steps.GenerateMeta(deps).Run(t.Context(), sc); err != nil {
		t.Fatalf("GenerateMeta: %v", err)
	}
	if !strings.Contains(recorder.last, "espresso | Shop") {
		t.Fatalf("the prompt does not carry the resolved title pattern:\n%s", recorder.last)
	}
	if !strings.Contains(recorder.last, "A short guide to espresso.") {
		t.Fatalf("the prompt does not carry the draft summary:\n%s", recorder.last)
	}
}

func TestGenerateMetaReportsWhatItCannotRead(t *testing.T) {
	t.Parallel()

	boom := errors.New(errors.External, "the database is busy")

	cases := []struct {
		name string
		deps func(steps.Deps) steps.Deps
		with func(*run.StepContext)
		want errors.Code
	}{
		{
			name: "there is no draft",
			with: func(sc *run.StepContext) { sc.Artifacts = map[run.ArtifactKind]run.Artifact{} },
			want: errors.Invalid,
		},
		{
			name: "the site is gone",
			deps: func(d steps.Deps) steps.Deps { d.Sites = siteStub{err: boom}; return d },
			want: errors.External,
		},
		{
			name: "the profile cannot be resolved",
			deps: func(d steps.Deps) steps.Deps { d.Profiles = profileStub{err: boom}; return d },
			want: errors.External,
		},
		{
			name: "the model fails",
			deps: func(d steps.Deps) steps.Deps { d.LLM = llmStub{err: boom}; return d },
			want: errors.External,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sc, deps := metaContext(t, `{"title":"x"}`)
			if tc.deps != nil {
				deps = tc.deps(deps)
			}
			if tc.with != nil {
				tc.with(sc)
			}
			if _, err := steps.GenerateMeta(deps).Run(t.Context(), sc); !errors.IsCode(err, tc.want) {
				t.Fatalf("code = %q, want %q (err %v)", errors.CodeOf(err), tc.want, err)
			}
		})
	}
}
