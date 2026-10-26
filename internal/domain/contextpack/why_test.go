package contextpack_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func openFinding(id string, code finding.Code, subject string) finding.Finding {
	return finding.Finding{ID: id, Code: code, Status: finding.Open, Subject: json.RawMessage(subject)}
}

func TestAnEmptyParentAlwaysSaysWhy(t *testing.T) {
	t.Parallel()

	live := pagemap.Page{ID: "p", Path: "/serums/", WPID: new(int64(7)), Status: pagemap.StatusPublished}
	planned := pagemap.Page{ID: "p", Path: "/serums/", Status: pagemap.StatusPlanned}
	front := pagemap.Page{ID: "f", Path: pagemap.RootPath, WPID: new(int64(1)), Status: pagemap.StatusPublished}
	archived := pagemap.Page{ID: "a", Path: "/old-serums/", WPID: new(int64(3)), Status: pagemap.StatusArchived}

	cases := []struct {
		name     string
		evidence contextpack.ParentEvidence
		want     contextpack.ParentWhy
	}{
		{
			name:     "the front page is the root by design",
			evidence: contextpack.ParentEvidence{Page: front, Rendered: true},
			want:     contextpack.ParentWhy{Reason: contextpack.WhyRoot},
		},
		{
			name:     "a parent that was archived leaves the page at the top",
			evidence: contextpack.ParentEvidence{Page: live, Parent: &archived, Rendered: true},
			want:     contextpack.ParentWhy{Reason: contextpack.WhyParentArchived, Parent: "/old-serums/"},
		},
		{
			name: "a loop the trails made was broken at the page",
			evidence: contextpack.ParentEvidence{Page: live, Rendered: true, Findings: []finding.Finding{
				openFinding("f1", finding.ParentCycle, `{"paths":["/serums/","/oils/"]}`),
			}},
			want: contextpack.ParentWhy{Reason: contextpack.WhyCycleBroken, Loop: []string{"/serums/", "/oils/"}, FindingID: "f1"},
		},
		{
			name: "a crumb two pages answer",
			evidence: contextpack.ParentEvidence{Page: live, Rendered: true, Findings: []finding.Finding{
				openFinding("f2", finding.ParentAmbiguous, `{"name":"Serums","candidates":["/serums-a/","/serums-b/"]}`),
			}},
			want: contextpack.ParentWhy{
				Reason: contextpack.WhyCrumbAmbiguous, Crumb: "Serums", Candidates: []string{"/serums-a/", "/serums-b/"}, FindingID: "f2",
			},
		},
		{
			name: "a crumb no page answers, by its link",
			evidence: contextpack.ParentEvidence{Page: live, Rendered: true, Findings: []finding.Finding{
				openFinding("f3", finding.ParentUnresolved, `{"name":"Skin care","path":"/skin-care/"}`),
			}},
			want: contextpack.ParentWhy{Reason: contextpack.WhyCrumbUnresolved, Crumb: "/skin-care/", FindingID: "f3"},
		},
		{
			name: "a crumb no page answers, by its name",
			evidence: contextpack.ParentEvidence{Page: live, Rendered: true, Findings: []finding.Finding{
				openFinding("f4", finding.ParentUnresolved, `{"name":"Skin care","path":""}`),
			}},
			want: contextpack.ParentWhy{Reason: contextpack.WhyCrumbUnresolved, Crumb: "Skin care", FindingID: "f4"},
		},
		{
			name:     "a planned page waits for the workbook's parent or the site",
			evidence: contextpack.ParentEvidence{Page: planned},
			want:     contextpack.ParentWhy{Reason: contextpack.WhyPlanned},
		},
		{
			name: "a render the observation could not fetch",
			evidence: contextpack.ParentEvidence{Page: live, Rendered: true, Findings: []finding.Finding{
				openFinding("f5", finding.RenderUnreachable, `{"path":"/serums/","status":503}`),
			}},
			want: contextpack.ParentWhy{Reason: contextpack.WhyRenderFailed, FindingID: "f5"},
		},
		{
			name:     "a page on the site no walk has rendered yet",
			evidence: contextpack.ParentEvidence{Page: live},
			want:     contextpack.ParentWhy{Reason: contextpack.WhyNotDerived},
		},
		{
			name:     "a rendered page with no trail, no WordPress parent and a top-level address",
			evidence: contextpack.ParentEvidence{Page: live, Rendered: true},
			want:     contextpack.ParentWhy{Reason: contextpack.WhyNoSignal},
		},
		{
			name: "a settled finding says nothing",
			evidence: contextpack.ParentEvidence{Page: live, Rendered: true, Findings: []finding.Finding{
				{ID: "f6", Code: finding.ParentUnresolved, Status: finding.Resolved, Subject: json.RawMessage(`{"name":"x","path":"/x/"}`)},
			}},
			want: contextpack.ParentWhy{Reason: contextpack.WhyNoSignal},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := contextpack.WhyNoParent(tc.evidence)
			if got.Reason != tc.want.Reason || got.Crumb != tc.want.Crumb || got.Parent != tc.want.Parent ||
				got.FindingID != tc.want.FindingID || !slices.Equal(got.Candidates, tc.want.Candidates) ||
				!slices.Equal(got.Loop, tc.want.Loop) {
				t.Fatalf("WhyNoParent = %+v, want %+v", got, tc.want)
			}
		})
	}
}
