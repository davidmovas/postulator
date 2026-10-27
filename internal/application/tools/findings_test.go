package tools_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func seedFinding(t *testing.T, seeded fixture, siteID string, code finding.Code, subject string) finding.Finding {
	t.Helper()

	held := finding.Finding{
		ID: id.New(), SiteID: siteID, Code: code, SubjectKey: finding.SubjectSite + ":" + string(code),
		Subject: json.RawMessage(subject), CreatedAt: stamp, UpdatedAt: stamp,
	}
	if siteID == seeded.site {
		pageID := seeded.page
		held.PageID, held.SubjectKey = &pageID, finding.SubjectPage(pageID)+":"+string(code)
	}
	record, err := finding.New(held)
	if err != nil {
		t.Fatalf("finding.New: %v", err)
	}
	if err = sqlite.NewFindingRepo(seeded.store).Upsert(t.Context(), record); err != nil {
		t.Fatalf("seed the finding: %v", err)
	}
	return record
}

func callFindings(t *testing.T, registry *tools.Registry, binding tools.Binding, args string) any {
	t.Helper()

	out, err := registry.Call(t.Context(), binding, "findings", json.RawMessage(args))
	if err != nil {
		t.Fatalf("findings %s: %v", args, err)
	}
	return out
}

func entityType(t *testing.T, seeded fixture) string {
	t.Helper()

	listed, err := seeded.types.List(t.Context(), pagetypes.ListRequest{SiteID: seeded.site})
	if err != nil {
		t.Fatalf("list the page types: %v", err)
	}
	for i := range listed.Types {
		if listed.Types[i].Preset == "guide" {
			return listed.Types[i].ID
		}
	}
	t.Fatal("the site has no entity type")
	return ""
}

func TestTheAgentListsAndResolvesFindings(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeConfirm
	typed := seedFinding(t, seeded, seeded.site, finding.TechnicalDetected, `{"reason":"shortcode"}`)
	changed := seedFinding(t, seeded, seeded.site, finding.FieldChangedOnSite, `{"key":"seo_description","site":"Theirs"}`)

	listed, ok := callFindings(t, registry, binding, `{"op":"list"}`).(tools.FindingsResponse)
	if !ok || listed.List == nil || len(listed.List.Items) != 2 {
		t.Fatalf("list = %+v, want both open findings at once, read without a card", listed)
	}
	first, ok := callFindings(t, registry, binding, `{"op":"list","limit":1}`).(tools.FindingsResponse)
	if !ok || len(first.List.Items) != 1 || !first.List.HasMore {
		t.Fatalf("the first page = %+v, want one finding and a cursor", first.List)
	}
	coded, ok := callFindings(t, registry, binding, `{"op":"list","code":"technical_detected"}`).(tools.FindingsResponse)
	if !ok || len(coded.List.Items) != 1 || coded.List.Items[0].ID != typed.ID {
		t.Fatalf("by code = %+v", coded.List)
	}

	pageType := entityType(t, seeded)
	resolve := `{"op":"resolve","findingId":"` + typed.ID + `","option":"set_type","pageTypeId":"` + pageType + `"}`
	asked, ok := callFindings(t, registry, binding, resolve).(tools.Confirmation)
	if !ok || !strings.Contains(asked.Summary, "/coffee/") || !strings.Contains(asked.Summary, "nothing is written to the site") {
		t.Fatalf("set_type in confirm mode = %+v, want a card that names the page and says the site is untouched", asked)
	}
	binding.Approved = true
	callFindings(t, registry, binding, resolve)
	settled, err := sqlite.NewFindingRepo(seeded.store).Get(t.Context(), typed.ID)
	if err != nil || settled.Status != finding.Resolved || settled.ResolvedBy != finding.ResolvedByAgent {
		t.Fatalf("the typed finding = %+v, %v, want it resolved by the agent", settled, err)
	}

	binding.Approved = false
	rewrite := `{"op":"resolve","findingId":"` + changed.ID + `","option":"rewrite_field"}`
	answered, ok := callFindings(t, registry, binding, rewrite).(tools.FindingsResponse)
	if !ok || answered.Resolve == nil || answered.Resolve.Start == nil {
		t.Fatalf("a run option in confirm mode = %+v, want the start request at once, with no card", answered)
	}
	if start := answered.Resolve.Start; len(start.Fields) != 1 || start.Fields[0] != "seo_description" ||
		len(start.Rewrite) != 1 || start.Rewrite[0] != seeded.page {
		t.Errorf("the start request = %+v, want the field and the page whose text it rewrites", start)
	}
}

func TestTheAgentResolvesAFindingByTheConversationsMode(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeAutonomous
	held := seedFinding(t, seeded, seeded.site, finding.TechnicalDetected, `{"reason":"shortcode"}`)
	if _, ok := callFindings(t, registry, binding, `{"op":"resolve","findingId":"`+held.ID+`","option":"ignore"}`).(tools.FindingsResponse); !ok {
		t.Fatal("an autonomous resolution waited for a person, want it made at once")
	}
	settled, err := sqlite.NewFindingRepo(seeded.store).Get(t.Context(), held.ID)
	if err != nil || settled.Status != finding.Dismissed || settled.ResolvedBy != finding.ResolvedByAgent {
		t.Fatalf("the finding = %+v, %v, want it dismissed by the agent", settled, err)
	}
}

func TestTheAgentCannotReachAnotherSitesFinding(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeAutonomous
	other := sqlitetest.Site(t, seeded.store, "elsewhere")
	held := seedFinding(t, seeded, other.ID, finding.TechnicalDetected, `{"reason":"shortcode"}`)
	_, err := registry.Call(t.Context(), binding, "findings", json.RawMessage(`{"op":"resolve","findingId":"`+held.ID+`","option":"ignore"}`))
	if !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("resolving another site's finding = %v, want NotFound", err)
	}
}
