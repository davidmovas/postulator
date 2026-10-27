//go:build !race

package scale_test

import (
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/ncruces/go-sqlite3"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
)

var largeTables = map[string]bool{
	"pages": true, "page_links": true, "page_fields": true, "page_bodies": true, "page_renders": true, "page_identity": true,
	"page_terms": true, "page_blocks_written": true, "page_map": true, "findings": true, "entities": true, "entity_anchors": true,
	"edges": true, "run_items": true, "run_events": true,
}

var byDesign = []struct {
	reads  []string
	reason string
}{
	{
		reads:  []string{"FROM page_map WHERE", "trail >= ?"},
		reason: "a map search under a node sorts that node's subtree by path, never the site",
	},
	{
		reads:  []string{"FROM findings AS f JOIN page_map AS m"},
		reason: "the findings list under a node or of a page type sorts the findings that subtree or type holds before it pages them",
	},
	{
		reads:  []string{"FROM run_items i JOIN runs r ON r.id = i.run_id"},
		reason: "the engine's sweep orders the run items that wait to run or for a parent, which the active runs bound, never the site",
	},
}

var (
	aliased = regexp.MustCompile(`(?i)\b(?:from|join)\s+([a-z_]+)\s+(?:as\s+)?([a-z]+)\b`)
	scanned = regexp.MustCompile(`^SCAN ([a-z_]+)`)
	sorted  = regexp.MustCompile(`^(?:SCAN|SEARCH) ([a-z_]+)`)
)

func tablesOf(statement string) map[string]string {
	out := map[string]string{}
	for _, match := range aliased.FindAllStringSubmatch(statement, -1) {
		out[strings.ToLower(match[2])] = strings.ToLower(match[1])
	}
	return out
}

func explainable(statement string) bool {
	head := strings.ToUpper(strings.Fields(statement + " x")[0])
	return slices.Contains([]string{"SELECT", "INSERT", "UPDATE", "DELETE", "WITH"}, head)
}

func explainAll(t *testing.T, path string, statements []string) map[string][]string {
	t.Helper()

	conn, err := sqlite3.OpenFlags(path, sqlite3.OPEN_READONLY)
	if err != nil {
		t.Fatalf("open the store to explain its statements: %v", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			t.Errorf("close the explaining store: %v", closeErr)
		}
	}()
	out := map[string][]string{}
	for _, statement := range statements {
		if explainable(statement) {
			out[statement] = explain(t, conn, statement)
		}
	}
	return out
}

func explain(t *testing.T, conn *sqlite3.Conn, statement string) []string {
	t.Helper()

	stmt, _, err := conn.Prepare("EXPLAIN QUERY PLAN " + statement)
	if err != nil {
		t.Fatalf("explain %q: %v", statement, err)
	}
	defer func() {
		if closeErr := stmt.Close(); closeErr != nil {
			t.Errorf("close the plan of %q: %v", statement, closeErr)
		}
	}()
	details := make([]string, 0)
	for stmt.Step() {
		details = append(details, stmt.ColumnText(3))
	}
	if err = stmt.Err(); err != nil {
		t.Fatalf("read the plan of %q: %v", statement, err)
	}
	return details
}

func large(statement, table string) (string, bool) {
	if named, isAlias := tablesOf(statement)[table]; isAlias {
		table = named
	}
	return table, largeTables[table]
}

func faultsOf(statement string, details []string) []string {
	out := make([]string, 0)
	paged := strings.Contains(compact(statement), " LIMIT ")
	read := ""
	for _, detail := range details {
		if match := scanned.FindStringSubmatch(detail); match != nil {
			if table, big := large(statement, match[1]); big {
				out = append(out, "a scan of "+table+" ("+detail+")")
			}
		}
		if match := sorted.FindStringSubmatch(detail); match != nil && read == "" {
			read = match[1]
		}
		if paged && strings.HasPrefix(detail, "USE TEMP B-TREE FOR ORDER BY") && read != "" {
			if table, big := large(statement, read); big {
				out = append(out, "a page sorted out of every matching row of "+table)
			}
		}
	}
	return out
}

func designed(statement string) (string, bool) {
	text := compact(statement)
	for _, held := range byDesign {
		matched := true
		for _, read := range held.reads {
			matched = matched && strings.Contains(text, read)
		}
		if matched {
			return held.reason, true
		}
	}
	return "", false
}

func compact(statement string) string {
	return strings.Join(strings.Fields(statement), " ")
}

func (s *scaled) scansNothingLarge(t *testing.T) {
	t.Helper()

	statements := make([]string, 0, len(s.traced))
	for statement := range s.traced {
		statements = append(statements, statement)
	}
	slices.Sort(statements)
	for statement, details := range explainAll(t, s.path, statements) {
		faults := faultsOf(statement, details)
		if len(faults) == 0 {
			continue
		}
		if reason, held := designed(statement); held {
			t.Logf("by design, %s: %s", reason, compact(statement))
			continue
		}
		t.Errorf("%s: %v", compact(statement), faults)
	}
}

func TestThePlanCheckSeesWhatALargeReadScans(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "postulator.db")
	store, err := sqlite.Open(sqlite.Config{Path: path})
	if err != nil {
		t.Fatalf("open the store: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close the store: %v", closeErr)
		}
	})
	cases := []struct {
		statement string
		faults    int
	}{
		{statement: "SELECT id FROM pages WHERE title = ?", faults: 1},
		{statement: "SELECT f.id FROM findings AS f WHERE f.subject = ?", faults: 1},
		{statement: "SELECT id FROM pages WHERE id = ?", faults: 0},
		{statement: "SELECT id FROM pages WHERE site_id = ? ORDER BY h1 LIMIT 10", faults: 1},
		{statement: "SELECT id FROM pages WHERE site_id = ? ORDER BY title, id LIMIT 10", faults: 0},
		{statement: "SELECT id FROM findings WHERE site_id = ? AND code = ? ORDER BY created_at, id LIMIT 21", faults: 0},
		{statement: "SELECT key, value FROM settings", faults: 0},
	}
	statements := make([]string, 0, len(cases))
	for _, tc := range cases {
		statements = append(statements, tc.statement)
	}
	plans := explainAll(t, path, statements)
	for _, tc := range cases {
		if got := faultsOf(tc.statement, plans[tc.statement]); len(got) != tc.faults {
			t.Errorf("%s plans %v and faults %v, want %d faults", tc.statement, plans[tc.statement], got, tc.faults)
		}
	}
}
