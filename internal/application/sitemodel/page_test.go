package sitemodel_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestThePageModelAnswersWhatTheAgentAsksOfAPage(t *testing.T) {
	t.Parallel()

	m := site1()
	theirs, ours := "<p>The owner's own opening.</p>", "<p>"+strings.Repeat("Our long paragraph on storage. ", 40)+"</p>"
	raw := theirs + "\n\n" + ours
	m.bodies[0].Raw, m.bodies[0].Hash = raw, block.BodyHash(raw)
	m.written = []block.Written{{PageID: "bpc", BlockHash: block.Parse(raw).Blocks[1].Hash, RunID: "run-1"}}
	m.fields["bpc"] = []field.Value{written(field.NewValue("bpc", field.Excerpt, "Our excerpt.", stamp), "run-1")}
	m.pages[2].ParentMethod = pagemap.MethodHref
	m.pages[2].TypeSource, m.pages[2].TypeReason = pagemap.TypeSourceAuto, pagemap.TypeByDefault
	m.findings = append(m.findings, finding.Finding{
		ID: "f3", SiteID: siteID, PageID: new("bpc"), Code: finding.FieldChangedOnSite, Severity: finding.Warning,
		Options: finding.OptionsOf(finding.FieldChangedOnSite), Status: finding.Open,
	})

	got, err := m.service().Page(t.Context(), sitemodel.PageRequest{SiteID: siteID, Path: "/bpc-157/"})
	if err != nil {
		t.Fatalf("Page: %v", err)
	}
	if got.Identity.WPID == nil || *got.Identity.WPID != 3 || got.Identity.URL != "https://shop.example.com/bpc-157/" {
		t.Errorf("identity = %+v, want the page's WordPress id and address", got.Identity)
	}
	if got.Type.Name == "" || got.Type.Source == "" {
		t.Errorf("type = %+v, want its name and where it came from", got.Type)
	}
	if got.Parent.Reason != pagemap.ParentJSONLD || got.Parent.Method != pagemap.MethodHref || len(got.Parent.Trail) == 0 {
		t.Errorf("parent = %+v, want the trail with its reason and method", got.Parent)
	}
	if got.Body.Blocks != 2 || got.Body.Ours != 1 || got.Body.Client != 1 || got.Body.Hash != block.BodyHash(raw) {
		t.Errorf("body = %+v, want two blocks, one of each owner, and the body's hash", got.Body)
	}
	excerpt := slices.IndexFunc(got.Fields, func(row sitemodel.PageField) bool { return row.Key == string(field.Excerpt) })
	if excerpt < 0 || got.Fields[excerpt].Owner != sitemodel.OwnerOurs || got.Fields[excerpt].Base != "Our excerpt." ||
		got.Fields[excerpt].Hash != field.Hash("Our excerpt.") {
		t.Errorf("fields = %+v, want the excerpt with its base, hash and owner", got.Fields)
	}
	for _, row := range got.Fields {
		if row.Key == string(field.Body) && strings.Contains(row.Current, "Our long paragraph on storage. Our long paragraph on storage. Our long") {
			t.Errorf("the body field carries %d bytes, want it summed up by the body, not repeated whole", len(row.Current))
		}
	}
	if len(got.Findings) != 1 || got.Findings[0].ID != "f3" || len(got.Findings[0].Options) != 3 {
		t.Errorf("findings = %+v, want the open one with its id and options", got.Findings)
	}
	if got.Paged["fields"] != "pages_fields get" || got.Paged["blocks"] != "pages_blocks list" || got.Paged["findings"] != "findings list" {
		t.Errorf("paged = %v, want each long list to name the op that pages it", got.Paged)
	}
	if len(got.Siblings)+len(got.Children) == 0 && got.Entity == nil {
		t.Errorf("the page model lost the neighborhood the page view carried: %+v", got)
	}
	if _, err := m.service().Page(t.Context(), sitemodel.PageRequest{SiteID: "elsewhere", PageID: "bpc"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("another site's page = %v, want NotFound", err)
	}
}
