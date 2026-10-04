package steps_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	storedValidation = `{"pageId":"page-child","score":0.7,` +
		`"compliance":{"items":[{"severity":"error","code":"target_missing","message":"no link"}],"score":0.75},` +
		`"structure":{"items":[{"severity":"warn","code":"thin","message":"short"}],"score":0.95}}`
	storedJudge  = `{"score":0.5,"issues":["thin"],"suggestions":["add an example"]}`
	storedRelink = `{"neighbors":[],"findings":[{"severity":"warn","code":"relink_conflict","message":"changed"}],` +
		`"linked":0,"conflicts":1}`
	storedPublish = `{"wpId":7,"url":"/coffee/espresso/","status":"draft"}`
	storedSync    = `{"wpId":7,"status":"draft","links":2,"source":"plugin"}`
	storedImages  = `{"featuredId":3,"images":[{"role":"featured","wpId":3}]}`
)

func runReport(t *testing.T, artifacts map[run.ArtifactKind][]byte) steps.FinalReport {
	t.Helper()

	result, err := steps.Report(unitDeps()).Run(t.Context(), unitContext(t, artifacts))
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	if len(result.Artifacts) != 1 || result.Artifacts[0].Kind != run.ArtifactFinalReport {
		t.Fatalf("Report produced %+v", result.Artifacts)
	}

	var report steps.FinalReport
	if err = json.Unmarshal(result.Artifacts[0].Blob, &report); err != nil {
		t.Fatalf("decode the final report: %v", err)
	}
	return report
}

func TestReportAggregatesEveryArtifact(t *testing.T) {
	t.Parallel()

	report := runReport(t, map[run.ArtifactKind][]byte{
		run.ArtifactValidationReport: []byte(storedValidation),
		run.ArtifactJudgeReport:      []byte(storedJudge),
		run.ArtifactPublishResult:    []byte(storedPublish),
		run.ArtifactRelinkResult:     []byte(storedRelink),
		run.ArtifactSyncResult:       []byte(storedSync),
		run.ArtifactImages:           []byte(storedImages),
	})

	if report.PageID != "page-child" || report.Path != "/coffee/espresso/" {
		t.Fatalf("report = %+v", report)
	}
	if report.Validation == nil || report.Judge == nil || report.Publish == nil ||
		report.Relink == nil || report.Sync == nil || report.Images == nil {
		t.Fatalf("report = %+v", report)
	}
	if report.Score == nil || *report.Score != 0.5 {
		t.Errorf("score = %v, want the lower of the validation and the judge", report.Score)
	}
	if len(report.Findings) != 3 {
		t.Fatalf("findings = %+v", report.Findings)
	}
	if report.Errors != 1 || report.Warnings != 2 {
		t.Errorf("errors = %d, warnings = %d", report.Errors, report.Warnings)
	}
}

func TestReportNoticesWhatThePageLacks(t *testing.T) {
	t.Parallel()

	validation := func(items, decisions string) []byte {
		return []byte(`{"pageId":"page-child","score":0.9,"compliance":{"items":[` + items + `],"score":0.9},` +
			`"structure":{"items":[],"score":1},"links":{"placed":[],"missing":[],"decisions":[` + decisions + `]}}`)
	}
	missing := `{"severity":"warn","code":"target_missing","message":"the page does not link to /a/"}`
	unpublished := `{"severity":"warn","code":"target_not_published","message":"the page links to /b/"}`
	capped := `{"target":{"url":"/a/"},"outcome":"cap_reached","detail":"spent"}`

	cases := []struct {
		name      string
		artifacts map[run.ArtifactKind][]byte
		want      string
	}{
		{
			name:      "a page that lacks nothing",
			artifacts: map[run.ArtifactKind][]byte{run.ArtifactValidationReport: validation("", "")},
		},
		{
			name:      "one owed link is missing",
			artifacts: map[run.ArtifactKind][]byte{run.ArtifactValidationReport: validation(missing, "")},
			want:      "1 owed link is missing",
		},
		{
			name: "the budget is spent",
			artifacts: map[run.ArtifactKind][]byte{
				run.ArtifactValidationReport: validation(missing+","+missing, capped),
			},
			want: "2 owed links are missing, the link budget of the page is spent",
		},
		{
			name:      "a link leads to a page not on the site",
			artifacts: map[run.ArtifactKind][]byte{run.ArtifactValidationReport: validation(unpublished, "")},
			want:      "1 link leads to a page that is not on the site yet",
		},
		{
			name: "a neighbor could not link back",
			artifacts: map[run.ArtifactKind][]byte{
				run.ArtifactRelinkResult: []byte(`{"neighbors":[],"findings":[{"severity":"warn",` +
					`"code":"neighbor_link_missing","message":"no room"}],"missing":1}`),
			},
			want: "1 neighbor could not link to this page",
		},
		{
			name: "fewer images than asked for",
			artifacts: map[run.ArtifactKind][]byte{
				run.ArtifactImages: []byte(`{"wanted":3,"images":[{"role":"featured","wpId":3}],"findings":[]}`),
			},
			want: "1 of 3 images placed",
		},
		{
			name: "all of it",
			artifacts: map[run.ArtifactKind][]byte{
				run.ArtifactValidationReport: validation(missing+","+unpublished, ""),
				run.ArtifactImages:           []byte(`{"wanted":1,"images":[],"findings":[]}`),
			},
			want: "1 owed link is missing; 1 link leads to a page that is not on the site yet; 0 of 1 images placed",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := steps.Report(unitDeps()).Run(t.Context(), unitContext(t, tc.artifacts))
			if err != nil {
				t.Fatalf("Report: %v", err)
			}
			if result.Notice != tc.want {
				t.Fatalf("notice = %q, want %q", result.Notice, tc.want)
			}
			var report steps.FinalReport
			if err = json.Unmarshal(result.Artifacts[0].Blob, &report); err != nil {
				t.Fatalf("decode the final report: %v", err)
			}
			if report.Notice != tc.want {
				t.Fatalf("report notice = %q, want %q", report.Notice, tc.want)
			}
			if tc.want != "" && !strings.Contains(result.Message, tc.want) {
				t.Fatalf("message = %q, want the notice in it", result.Message)
			}
		})
	}
}

func TestReportCarriesWhatThePublishAndTheReadBackFound(t *testing.T) {
	t.Parallel()

	report := runReport(t, map[run.ArtifactKind][]byte{
		run.ArtifactPublishResult: []byte(`{"wpId":7,"url":"/espresso/","status":"draft",` +
			`"findings":[{"severity":"warn","code":"seo_meta_skipped","message":"no plugin"}]}`),
		run.ArtifactSyncResult: []byte(`{"wpId":7,"status":"draft","links":2,"source":"plugin",` +
			`"findings":[{"severity":"warn","code":"plan_not_kept","message":"the h1 differs"}]}`),
	})

	codes := make([]string, 0, len(report.Findings))
	for _, finding := range report.Findings {
		codes = append(codes, finding.Code)
	}
	if !slices.Contains(codes, steps.CodeSEOMetaSkipped) || !slices.Contains(codes, steps.CodePlanNotKept) {
		t.Fatalf("the report carries %v, want what the publish and the read back found", codes)
	}
	if report.Warnings != 2 {
		t.Errorf("warnings = %d, want both counted", report.Warnings)
	}
}

func TestReportSurvivesAnEmptyItem(t *testing.T) {
	t.Parallel()

	report := runReport(t, nil)
	if report.Score != nil {
		t.Fatalf("score = %v, want none: nothing scored this page", *report.Score)
	}
	if len(report.Findings) != 0 {
		t.Fatalf("findings = %+v", report.Findings)
	}
	if report.Validation != nil || report.Judge != nil || report.Publish != nil {
		t.Fatalf("report = %+v", report)
	}
}

func TestReportNamesEveryArtifactRetentionTookAway(t *testing.T) {
	t.Parallel()

	purged := map[run.ArtifactKind]run.Artifact{
		run.ArtifactValidationReport: {Kind: run.ArtifactValidationReport, Purged: true},
		run.ArtifactJudgeReport:      {Kind: run.ArtifactJudgeReport, Purged: true},
		run.ArtifactPublishResult:    {Kind: run.ArtifactPublishResult, Blob: []byte(storedPublish)},
	}

	sc := unitContext(t, nil)
	sc.Artifacts = purged

	result, err := steps.Report(unitDeps()).Run(t.Context(), sc)
	if err != nil {
		t.Fatalf("Report: %v", err)
	}

	var report steps.FinalReport
	if err = json.Unmarshal(result.Artifacts[0].Blob, &report); err != nil {
		t.Fatalf("decode the final report: %v", err)
	}
	if report.Score != nil {
		t.Fatalf("score = %v, want none: the validation report it would come from is gone", *report.Score)
	}

	named := make([]string, 0, len(report.Findings))
	for _, finding := range report.Findings {
		if finding.Code != steps.CodeArtifactPurged {
			continue
		}
		if finding.Severity != content.SeverityWarn {
			t.Errorf("finding = %+v, want a warning", finding)
		}
		kind, ok := finding.Details["kind"].(string)
		if !ok {
			t.Fatalf("the finding %+v does not name the artifact it is about", finding)
		}
		named = append(named, kind)
	}
	slices.Sort(named)
	if !slices.Equal(named, []string{"judge_report", "validation_report"}) {
		t.Fatalf("the report names %v as gone, want both sections retention took", named)
	}
	if report.Warnings != 2 {
		t.Errorf("warnings = %d, want one per section that is gone", report.Warnings)
	}
}

func TestReportRefusesAnUnreadableArtifact(t *testing.T) {
	t.Parallel()

	_, err := steps.Report(unitDeps()).Run(t.Context(), unitContext(t, map[run.ArtifactKind][]byte{
		run.ArtifactValidationReport: []byte(`not json`),
	}))
	if !errors.IsCode(err, errors.Internal) {
		t.Fatalf("code = %q, want %q (err %v)", errors.CodeOf(err), errors.Internal, err)
	}
}
