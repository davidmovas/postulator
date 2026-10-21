//go:build e2e

package e2e_test

import (
	"bytes"
	"cmp"
	"encoding/json"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	domaingraph "github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/render"
	domainsite "github.com/davidmovas/postulator/internal/domain/site"
)

const (
	captureSwitch = "E2E_CAPTURE"
	captureTime   = "2026-10-19T09:00:00Z"
	captureBase   = "https://stand.example"
	zeroTime      = "0001-01-01T00:00:00Z"
)

var (
	idPattern    = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	stampPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})?$`)
)

func TestTheStandModelIsCaptured(t *testing.T) {
	if os.Getenv(captureSwitch) != "1" {
		t.Skipf("set %s=1 to capture the stand's model for the pack goldens; run it first on a fresh stand", captureSwitch)
	}
	live := newSite(t)
	requirePlugin(t, live.env, true)
	stand := writerStandOf(t, live.env)

	core := openCore(t)
	owner, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name: "Captured Shop", BaseURL: live.env.baseURL, Username: live.env.user, Password: live.env.pass, AllowInsecure: true,
	})
	if err != nil {
		t.Fatalf("create the site: %v", err)
	}
	siteID := owner.Site.ID
	runSync(t, core, siteID)
	seedStandGraph(t, core, siteID, stand.graph)

	rows := readStandRows(t, core, siteID)
	ids := rows.settle()
	encoded := rows.normalized(t, live.env, ids)

	path := filepath.Join(repoRoot(t), "internal", "application", "sitemodel", "testdata", live.env.stand, "site.json")
	if err = os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("make the directory of %s: %v", path, err)
	}
	if err = os.WriteFile(path, encoded, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	t.Logf("captured %d pages, %d entities, %d edges, %d types, %d bodies, %d renders, %d links and %d open findings "+
		"of the %s stand into %s", len(rows.pages), len(rows.entities), len(rows.edges), len(rows.types), len(rows.bodies),
		len(rows.renders), len(rows.links), len(rows.findings), live.env.stand, path)
}

type standRows struct {
	site     domainsite.Site
	terms    map[string][]pagemap.Term
	fields   map[string][]field.Value
	pages    []pagemap.Page
	entities []domaingraph.Entity
	edges    []domaingraph.Edge
	types    []pagetype.Type
	scopes   []field.Scope
	bodies   []block.Snapshot
	written  []block.Written
	renders  []render.Record
	links    []pagemap.PageLink
	findings []finding.Finding
}

func readStandRows(t *testing.T, core *app.Core, siteID string) standRows {
	t.Helper()

	ctx := t.Context()
	fail := func(what string, err error) {
		if err != nil {
			t.Fatalf("read the %s of the captured site: %v", what, err)
		}
	}
	var (
		rows standRows
		err  error
	)
	rows.site, err = sqlite.NewSiteRepo(core.Store).Get(ctx, siteID)
	fail("site", err)
	rows.pages, err = sqlite.NewPageRepo(core.Store).ListBySite(ctx, siteID)
	fail("pages", err)
	rows.entities, err = sqlite.NewEntityRepo(core.Store).ListBySite(ctx, siteID)
	fail("entities", err)
	rows.edges, err = sqlite.NewEdgeRepo(core.Store).ListBySite(ctx, siteID)
	fail("edges", err)
	rows.types, err = sqlite.NewPageTypeRepo(core.Store).ListBySite(ctx, siteID)
	fail("page types", err)
	rows.scopes, err = sqlite.NewFieldScopeRepo(core.Store).ListBySite(ctx, siteID)
	fail("field scopes", err)
	rows.terms, err = sqlite.NewPageIdentityRepo(core.Store).ListTermsBySite(ctx, siteID)
	fail("terms", err)
	rows.fields, err = sqlite.NewPageFieldRepo(core.Store).ListBySite(ctx, siteID)
	fail("fields", err)
	bodies := sqlite.NewPageBodyRepo(core.Store)
	snapshots, err := bodies.ListBySite(ctx, siteID)
	fail("bodies", err)
	rows.bodies = slices.DeleteFunc(snapshots, func(snapshot block.Snapshot) bool { return snapshot.Side != block.Current })
	rows.written = make([]block.Written, 0)
	for i := range rows.pages {
		written, writtenErr := bodies.ListWritten(ctx, rows.pages[i].ID)
		fail("written blocks", writtenErr)
		rows.written = append(rows.written, written...)
	}
	rows.renders, err = sqlite.NewPageRenderRepo(core.Store).ListBySite(ctx, siteID)
	fail("renders", err)
	rows.links, err = sqlite.NewPageLinkRepo(core.Store).ListBySite(ctx, siteID)
	fail("links", err)
	rows.findings, err = sqlite.NewFindingRepo(core.Store).ListBySite(ctx, siteID, finding.Open)
	fail("open findings", err)
	if len(rows.pages) == 0 {
		t.Fatal("the captured site holds no page")
	}
	rows.entities, rows.edges, rows.types, rows.scopes = orEmpty(rows.entities), orEmpty(rows.edges), orEmpty(rows.types), orEmpty(rows.scopes)
	rows.bodies, rows.renders, rows.links, rows.findings = orEmpty(rows.bodies), orEmpty(rows.renders), orEmpty(rows.links), orEmpty(rows.findings)
	if rows.terms == nil {
		rows.terms = map[string][]pagemap.Term{}
	}
	if rows.fields == nil {
		rows.fields = map[string][]field.Value{}
	}
	return rows
}

func orEmpty[T any](rows []T) []T {
	if rows == nil {
		return []T{}
	}
	return rows
}

func (rows *standRows) settle() map[string]string {
	ids := map[string]string{rows.site.ID: "site"}
	taken := make(map[string]int)
	name := func(raw, stable string) {
		taken[stable]++
		if taken[stable] > 1 {
			stable += "#" + strconv.Itoa(taken[stable])
		}
		ids[raw] = stable
	}
	for i := range rows.pages {
		name(rows.pages[i].ID, "page:"+rows.pages[i].Path)
	}
	for i := range rows.entities {
		name(rows.entities[i].ID, "entity:"+rows.entities[i].Name)
	}
	for i := range rows.edges {
		edge := &rows.edges[i]
		name(edge.ID, "edge:"+string(edge.Kind)+":"+ids[edge.FromEntityID]+">"+ids[edge.ToEntityID])
	}
	for i := range rows.types {
		name(rows.types[i].ID, "type:"+string(rows.types[i].Preset))
	}

	slices.SortStableFunc(rows.edges, func(a, b domaingraph.Edge) int { return strings.Compare(ids[a.ID], ids[b.ID]) })
	slices.SortStableFunc(rows.types, func(a, b pagetype.Type) int { return strings.Compare(ids[a.ID], ids[b.ID]) })
	slices.SortStableFunc(rows.scopes, func(a, b field.Scope) int {
		return cmp.Or(strings.Compare(ids[a.PageTypeID], ids[b.PageTypeID]), strings.Compare(string(a.Key), string(b.Key)))
	})
	slices.SortStableFunc(rows.bodies, func(a, b block.Snapshot) int { return strings.Compare(ids[a.PageID], ids[b.PageID]) })
	slices.SortStableFunc(rows.written, func(a, b block.Written) int {
		return cmp.Or(strings.Compare(ids[a.PageID], ids[b.PageID]), strings.Compare(a.BlockHash, b.BlockHash))
	})
	slices.SortStableFunc(rows.renders, func(a, b render.Record) int { return strings.Compare(ids[a.PageID], ids[b.PageID]) })
	slices.SortStableFunc(rows.links, func(a, b pagemap.PageLink) int { return compareLinks(ids, &a, &b) })
	numbered := make(map[string]int)
	for i := range rows.links {
		from := ids[rows.links[i].FromPageID]
		numbered[from]++
		ids[rows.links[i].ID] = "link:" + from + "#" + strconv.Itoa(numbered[from])
	}
	slices.SortStableFunc(rows.findings, func(a, b finding.Finding) int {
		return cmp.Or(strings.Compare(string(a.Code), string(b.Code)),
			strings.Compare(known(ids, a.SubjectKey), known(ids, b.SubjectKey)))
	})
	for i := range rows.findings {
		code := "finding:" + string(rows.findings[i].Code)
		numbered[code]++
		ids[rows.findings[i].ID] = code + "#" + strconv.Itoa(numbered[code])
	}
	return ids
}

func compareLinks(ids map[string]string, a, b *pagemap.PageLink) int {
	return cmp.Or(
		strings.Compare(ids[a.FromPageID], ids[b.FromPageID]),
		strings.Compare(string(a.Source), string(b.Source)),
		strings.Compare(a.Region, b.Region),
		cmp.Compare(positionOf(a.Position), positionOf(b.Position)),
		strings.Compare(a.ToURL, b.ToURL),
		strings.Compare(a.AnchorText, b.AnchorText),
		strings.Compare(string(a.Type), string(b.Type)),
		compareOurs(a.Ours, b.Ours),
	)
}

func positionOf(position *int) int {
	if position == nil {
		return -1
	}
	return *position
}

func compareOurs(a, b bool) int {
	switch {
	case a == b:
		return 0
	case a:
		return 1
	default:
		return -1
	}
}

func known(ids map[string]string, text string) string {
	return idPattern.ReplaceAllStringFunc(text, func(raw string) string {
		if stable, held := ids[raw]; held {
			return stable
		}
		return raw
	})
}

func (rows *standRows) normalized(t *testing.T, env environment, ids map[string]string) []byte {
	t.Helper()

	document := map[string]any{
		"site": rows.site, "pages": rows.pages, "entities": rows.entities, "edges": rows.edges, "types": rows.types,
		"scopes": rows.scopes, "terms": rows.terms, "fields": rows.fields, "bodies": rows.bodies, "written": rows.written,
		"renders": rows.renders, "links": rows.links, "findings": rows.findings,
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("encode the captured rows: %v", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var tree any
	if err = decoder.Decode(&tree); err != nil {
		t.Fatalf("decode the captured rows: %v", err)
	}

	parsed, err := url.Parse(env.baseURL)
	if err != nil {
		t.Fatalf("%s is not an address: %v", env.baseURL, err)
	}
	stable := normalizer{ids: ids, urls: strings.NewReplacer(
		env.baseURL, captureBase,
		strings.ReplaceAll(env.baseURL, "/", `\/`), strings.ReplaceAll(captureBase, "/", `\/`),
		url.QueryEscape(env.baseURL), url.QueryEscape(captureBase),
		parsed.Host, strings.TrimPrefix(captureBase, "https://"),
	)}
	tree = stable.value(tree)

	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err = encoder.Encode(tree); err != nil {
		t.Fatalf("encode the normalized rows: %v", err)
	}
	if stable.leftover > 0 {
		t.Logf("%d ids no captured row owns were numbered in the order they appear", stable.leftover)
	}
	return out.Bytes()
}

type normalizer struct {
	ids      map[string]string
	urls     *strings.Replacer
	leftover int
}

func (n *normalizer) value(held any) any {
	switch typed := held.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for _, key := range slices.Sorted(maps.Keys(typed)) {
			out[n.text(key)] = n.value(typed[key])
		}
		return out
	case []any:
		for i := range typed {
			typed[i] = n.value(typed[i])
		}
		return typed
	case string:
		return n.text(typed)
	default:
		return held
	}
}

func (n *normalizer) text(raw string) string {
	text := idPattern.ReplaceAllStringFunc(n.urls.Replace(raw), n.id)
	if text != zeroTime && stampPattern.MatchString(text) {
		return captureTime
	}
	return text
}

func (n *normalizer) id(raw string) string {
	if stable, held := n.ids[raw]; held {
		return stable
	}
	n.leftover++
	stable := "id:" + strconv.Itoa(n.leftover)
	n.ids[raw] = stable
	return stable
}
