//go:build e2e

package e2e_test

import (
	"maps"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/application/sites"
	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

const (
	planSheet   = "Topical plan"
	mapSheet    = "Site map"
	brokenSheet = "Broken"

	planRows = 61
	planHubs = 5
)

var messySheets = []string{"Site map", "Bikes", "Components", "Guides", "shop crawl (old)", "Broken", "Notes"}

var clientPrefixes = []string{"/electric-bikes", "/components", "/guides", "/maintenance", "/laws"}

func sample(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "samples", name)
}

func (s *site) clearUnder(t *testing.T, prefixes ...string) {
	t.Helper()

	listed := s.content(t)
	for i := range listed {
		if listed[i].Type != "page" {
			continue
		}
		under := false
		for _, prefix := range prefixes {
			if strings.HasPrefix(listed[i].Path, prefix) {
				under = true
			}
		}
		if !under {
			continue
		}
		s.call(t, http.MethodDelete, "/wp-json/wp/v2/pages/"+strconv.Itoa(listed[i].ID)+"?force=true",
			nil, http.StatusOK, nil)
	}
}

func planMapping(t *testing.T, core *app.Core, siteID string) imports.Mapping {
	t.Helper()

	seen, err := core.Imports.Inspect(t.Context(), imports.InspectRequest{
		SiteID: siteID, Path: sample(t, "entity-plan.xlsx"),
	})
	if err != nil {
		t.Fatalf("inspect entity-plan.xlsx: %v", err)
	}
	if seen.Rows != planRows {
		t.Fatalf("the plan carries %d rows, want %d", seen.Rows, planRows)
	}
	for _, field := range []string{"entity", "parent_entity", "path", "title", "primary_keyword", "keywords", "page_kind"} {
		if seen.Detected.Columns[field] == "" {
			t.Fatalf("%s is not detected in %v", field, seen.Headers)
		}
	}
	return seen.Detected
}

func assertEverySheetIsNamed(t *testing.T, core *app.Core, siteID string) {
	t.Helper()

	seen, err := core.Imports.Inspect(t.Context(), imports.InspectRequest{
		SiteID: siteID, Path: sample(t, "messy-sheets.xlsx"),
	})
	if err != nil {
		t.Fatalf("inspect messy-sheets.xlsx: %v", err)
	}

	named := make([]string, 0, len(seen.Sheets))
	for i := range seen.Sheets {
		if seen.Sheets[i].Name == "" {
			t.Fatalf("sheet %d of messy-sheets.xlsx has no name: %+v", i, seen.Sheets[i])
		}
		named = append(named, seen.Sheets[i].Name)
	}
	if !slices.Equal(named, messySheets) {
		t.Fatalf("the workbook names %v, want %v", named, messySheets)
	}
	for i := range seen.Sheets {
		if seen.Sheets[i].Rows == 0 {
			t.Fatalf("the sheet %q reports no rows", seen.Sheets[i].Name)
		}
	}
}

func assertTheIndentTreeIsRead(t *testing.T, core *app.Core, siteID string) {
	t.Helper()

	seen, err := core.Imports.Inspect(t.Context(), imports.InspectRequest{
		SiteID: siteID, Path: sample(t, "messy-sheets.xlsx"), Sheets: []string{mapSheet}, NoHeader: true,
	})
	if err != nil {
		t.Fatalf("inspect the %q sheet: %v", mapSheet, err)
	}
	if len(seen.Headers) == 0 {
		t.Fatalf("a sheet read with noHeader names no columns: %+v", seen)
	}

	mapping := imports.Mapping{
		Columns: map[string]string{"title": "D"},
		Options: imports.Options{
			Sheets:        []string{mapSheet},
			NoHeader:      true,
			IndentColumns: []string{"C", "D", "E"},
		},
	}
	preview, err := core.Imports.Preview(t.Context(), imports.PreviewRequest{
		SiteID: siteID, Path: sample(t, "messy-sheets.xlsx"), Mapping: mapping,
	})
	if err != nil {
		t.Fatalf("preview the indent tree: %v", err)
	}

	depths := make(map[int]int, 3)
	for i := range preview.Report.Pages {
		depths[strings.Count(strings.Trim(preview.Report.Pages[i].Path, "/"), "/")+1]++
	}
	for depth := 1; depth <= 3; depth++ {
		if depths[depth] == 0 {
			t.Fatalf("the indent tree read no page %d level(s) deep: %v", depth, depths)
		}
	}

	found := false
	for i := range preview.Report.Pages {
		if preview.Report.Pages[i].Path == "/electric-bikes/commuter/" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the indent tree did not build /electric-bikes/commuter/ from its columns; it read %d pages",
			len(preview.Report.Pages))
	}
}

func assertEveryPlantedFaultIsReported(t *testing.T, core *app.Core, siteID string) {
	t.Helper()

	seen, err := core.Imports.Inspect(t.Context(), imports.InspectRequest{
		SiteID: siteID, Path: sample(t, "messy-sheets.xlsx"), Sheets: []string{brokenSheet},
	})
	if err != nil {
		t.Fatalf("inspect the %q sheet: %v", brokenSheet, err)
	}
	if seen.Detected.Columns["path"] != "Page path" {
		t.Fatalf("the detector maps the path column to %q, want the sheet's own \"Page path\": %v",
			seen.Detected.Columns["path"], seen.Headers)
	}

	asDetected := seen.Detected
	asDetected.Options.Sheets = []string{brokenSheet}

	asTitles := seen.Detected
	asTitles.Options.Sheets = []string{brokenSheet}
	asTitles.Columns = maps.Clone(seen.Detected.Columns)
	asTitles.Columns["entity"] = seen.Detected.Columns["title"]

	readings := []struct {
		name    string
		mapping imports.Mapping
	}{{name: "detected", mapping: asDetected}, {name: "titles", mapping: asTitles}}

	reported := make(map[string][]imports.Finding, 8)
	for i := range readings {
		preview, previewErr := core.Imports.Preview(t.Context(), imports.PreviewRequest{
			SiteID: siteID, Path: sample(t, "messy-sheets.xlsx"), Mapping: readings[i].mapping,
		})
		if previewErr != nil {
			t.Fatalf("preview the %q sheet as %s: %v", brokenSheet, readings[i].name, previewErr)
		}
		for _, finding := range slices.Concat(preview.Report.Errors, preview.Report.Warnings) {
			reported[finding.Code] = append(reported[finding.Code], finding)
		}
	}

	for _, planted := range []struct {
		code string
		what string
	}{
		{code: string(imports.CodeBadPath), what: "the path with spaces in it"},
		{code: string(imports.CodeNoTarget), what: "the row that names nothing"},
		{code: string(imports.CodeUnknownParent), what: "the parent nobody declared"},
		{code: string(imports.CodeCycle), what: "the two rows that parent each other"},
		{code: string(imports.CodeSelfEdge), what: "the row that is its own parent"},
		{code: string(imports.CodeDuplicatePath), what: "the path that appears twice"},
	} {
		if len(reported[planted.code]) == 0 {
			t.Errorf("the %q sheet does not report %s as %q; it reported %v",
				brokenSheet, planted.what, planted.code, codesOf(reported))
		}
	}
	for _, finding := range reported[string(imports.CodeCycle)] {
		for _, named := range []string{"Loop A", "Loop B"} {
			if !strings.Contains(finding.Message, named) {
				t.Errorf("the cycle finding reads %q and names neither row a reader has to go and fix",
					finding.Message)
			}
		}
	}
	for _, code := range []string{string(imports.CodeBadPath), string(imports.CodeNoTarget),
		string(imports.CodeUnknownParent), string(imports.CodeSelfEdge), string(imports.CodeDuplicatePath)} {
		for _, finding := range reported[code] {
			if finding.Row == 0 {
				t.Errorf("the %q finding names no row: %+v", code, finding)
			}
		}
	}
}

func TestTheClientLoopFromTheSamples(t *testing.T) {
	live := newSite(t)
	requirePlugin(t, live.env, true)
	live.clearUnder(t, clientPrefixes...)

	script := &clientScript{}
	core := openCoreWith(t, fake.NewScripted(script.replies()...))
	owner, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name:          "Docker Shop",
		BaseURL:       live.env.baseURL,
		Username:      live.env.user,
		Password:      live.env.pass,
		AllowInsecure: true,
	})
	if err != nil {
		t.Fatalf("create the site: %v", err)
	}
	siteID := owner.Site.ID

	assertEverySheetIsNamed(t, core, siteID)
	assertTheIndentTreeIsRead(t, core, siteID)
	assertEveryPlantedFaultIsReported(t, core, siteID)

	mapping := planMapping(t, core, siteID)
	planned, err := core.Imports.Preview(t.Context(), imports.PreviewRequest{
		SiteID: siteID, Path: sample(t, "entity-plan.xlsx"), Mapping: mapping,
	})
	if err != nil {
		t.Fatalf("preview entity-plan.xlsx: %v", err)
	}

	filledIn := make([]string, 0, 2)
	for i := range planned.Report.Pages {
		if planned.Report.Pages[i].Generated {
			filledIn = append(filledIn, planned.Report.Pages[i].Path)
		}
	}

	applied, err := core.Imports.Apply(t.Context(), imports.ApplyRequest{
		SiteID: siteID, Path: sample(t, "entity-plan.xlsx"), Mapping: mapping,
	})
	if err != nil {
		t.Fatalf("apply entity-plan.xlsx: %v", err)
	}
	if len(applied.Report.Errors) != 0 {
		t.Fatalf("the plan reported %+v, want a clean sheet", applied.Report.Errors)
	}
	if applied.Counts.EntitiesCreated != planRows {
		t.Fatalf("the import wrote %d entities, want one per row of the sheet (%d)",
			applied.Counts.EntitiesCreated, planRows)
	}
	if applied.Counts.PagesCreated != planRows+len(filledIn) {
		t.Fatalf("the import wrote %d pages, want one per row of the sheet (%d) and one per path "+
			"the sheet skips over (%v)", applied.Counts.PagesCreated, planRows, filledIn)
	}
	if applied.Counts.EdgesCreated != planRows-planHubs {
		t.Fatalf("the import wrote %d edges, want one per row that names a parent (%d)",
			applied.Counts.EdgesCreated, planRows-planHubs)
	}
	t.Logf("the plan landed as %+v, filling in %v", applied.Counts, filledIn)

	approved := proposeAndApproveRelated(t, core, script, siteID)
	scores := recomputeScores(t, core, siteID)
	t.Logf("%d related edges were approved and %d entities were scored", approved, len(scores))
}

func proposeAndApproveRelated(t *testing.T, core *app.Core, script *clientScript, siteID string) int {
	t.Helper()

	wanted := []relatedPair{
		{from: "Commuter E-Bikes", to: "Folding E-Bikes", reason: "both are bought to ride to work"},
		{from: "E-Bike Batteries", to: "E-Bike Chargers", reason: "a battery is chosen with its charger"},
	}
	script.relate(wanted...)

	proposed, err := core.Graph.ProposeRelated(t.Context(), graph.ProposeRelatedRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("propose related edges: %v", err)
	}
	if len(proposed.Edges) != len(wanted) {
		t.Fatalf("the proposer wrote %d edges, want the %d the model named: %+v",
			len(proposed.Edges), len(wanted), proposed.Edges)
	}
	if proposed.Tokens == 0 {
		t.Fatal("the proposal cost no tokens, so nothing was asked of the model")
	}

	for i := range proposed.Edges {
		edge := proposed.Edges[i]
		if edge.Status != string(graphdomain.StatusProposed) {
			t.Fatalf("the proposed edge %s -> %s is %q, want proposed",
				edge.FromEntityID, edge.ToEntityID, edge.Status)
		}
		if strings.TrimSpace(edge.Reason) == "" {
			t.Fatalf("the proposed edge %s -> %s carries no reason", edge.FromEntityID, edge.ToEntityID)
		}
		if _, approveErr := core.Graph.ApproveEdge(t.Context(),
			graph.ApproveEdgeRequest{ID: edge.ID}); approveErr != nil {
			t.Fatalf("approve the edge %s: %v", edge.ID, approveErr)
		}
	}

	listed, err := core.Graph.ListEdges(t.Context(), graph.ListEdgesRequest{
		SiteID: siteID, Kind: string(graphdomain.EdgeRelated), ListRequest: dto.ListRequest{Limit: 100},
	})
	if err != nil {
		t.Fatalf("list the related edges: %v", err)
	}
	for i := range listed.Items {
		if listed.Items[i].Status != string(graphdomain.StatusApproved) {
			t.Fatalf("the related edge %s stayed %q after it was approved",
				listed.Items[i].ID, listed.Items[i].Status)
		}
	}
	if len(listed.Items) != len(wanted) {
		t.Fatalf("the site carries %d related edges, want the %d that were approved",
			len(listed.Items), len(wanted))
	}
	return len(listed.Items)
}

func recomputeScores(t *testing.T, core *app.Core, siteID string) map[string]float64 {
	t.Helper()

	recomputed, err := core.Graph.RecomputeScores(t.Context(), graph.RecomputeScoresRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("recompute the scores: %v", err)
	}
	if len(recomputed.Scores) == 0 {
		t.Fatal("the recompute scored no entity")
	}

	ranked := 0
	for _, score := range recomputed.Scores {
		if score > 0 {
			ranked++
		}
	}
	if ranked == 0 {
		t.Fatalf("every one of the %d scores is zero, so the graph carries no weight", len(recomputed.Scores))
	}
	return recomputed.Scores
}

func codesOf(reported map[string][]imports.Finding) []string {
	out := make([]string, 0, len(reported))
	for code := range reported {
		out = append(out, code)
	}
	slices.Sort(out)
	return out
}
