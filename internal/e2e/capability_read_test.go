//go:build e2e

package e2e_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/application/models"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/application/tools"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	readDefinition = "Healing peptides are the research peptides studied for tissue repair."
	readMapping    = "Capability workbook"
	readAnchor     = "the healing peptides we stock"
)

func TestTheAssistantReadsTheSite(t *testing.T) {
	c := openCapability(t, fake.NewScripted(), withTheTree)
	if _, err := c.core.Graph.AddEdge(t.Context(), graph.AddEdgeRequest{
		SiteID: c.siteID, FromEntityID: c.entities["BPC-157"], ToEntityID: c.entities["TB-500"], Kind: "related", Weight: 0.8,
		Reason: "Both are studied for tissue repair.",
	}); err != nil {
		t.Fatalf("relate BPC-157 and TB-500: %v", err)
	}
	definition, facts := readDefinition, []graph.Fact{{Label: "Storage", Value: "Frozen, sealed under nitrogen"}}
	if _, err := c.core.Graph.UpdateEntity(t.Context(), graph.UpdateEntityRequest{
		ID: c.entities["Healing Peptides"], Definition: &definition, Facts: &facts,
	}); err != nil {
		t.Fatalf("give Healing Peptides a card: %v", err)
	}
	if _, err := c.core.Graph.SetAnchors(t.Context(), graph.SetAnchorsRequest{
		EntityID: c.entities["Healing Peptides"], Anchors: []graph.Anchor{{Text: readAnchor, Source: "user"}},
	}); err != nil {
		t.Fatalf("give Healing Peptides an anchor: %v", err)
	}
	c.refresh(t)

	t.Run("01 the site at a glance", func(t *testing.T) { readsTheSiteAtAGlance(t, c) })
	t.Run("02 the page tree", func(t *testing.T) { readsThePageTree(t, c) })
	t.Run("03 the entity graph", func(t *testing.T) { readsTheEntityGraph(t, c) })
	t.Run("04 the tree against the graph where they agree", func(t *testing.T) { readsBothHierarchiesAgree(t, c) })
	t.Run("05 a page in full", func(t *testing.T) { readsAPageInFull(t, c) })
	t.Run("06 many pages at once", func(t *testing.T) { readsManyPagesAtOnce(t, c) })
	t.Run("07 links of a section", func(t *testing.T) { readsTheLinksOfASection(t, c) })
	t.Run("08 search", func(t *testing.T) { searchesByWords(t, c) })
	t.Run("09 findings, runs, costs and the import", func(t *testing.T) { readsFindingsRunsCostsAndImports(t, c) })
	t.Run("10 types, scopes, templates and rules", func(t *testing.T) { readsTypesScopesTemplatesAndRules(t, c) })
}

func livePages(c *capability) map[string]pages.Page {
	out := make(map[string]pages.Page, len(c.pages))
	for path := range c.pages {
		if c.pages[path].Status != string(pagemap.StatusArchived) {
			out[path] = c.pages[path]
		}
	}
	return out
}

func hubAt(hubs []contextpack.Hub, path string) (contextpack.Hub, bool) {
	for _, hub := range hubs {
		if hub.Path == path {
			return hub, true
		}
		if held, found := hubAt(hub.Children, path); found {
			return held, true
		}
	}
	return contextpack.Hub{}, false
}

func readsTheSiteAtAGlance(t *testing.T, c *capability) {
	conversation := c.conversation(t)
	answered := ask[tools.SitesResponse](t, c, conversation, "sites", map[string]any{"view": "overview"})
	overview := answered.Overview
	if overview == nil || overview.Site.ID != c.siteID {
		t.Fatalf("sites overview answered %+v, want this conversation's site", answered)
	}
	shape := overview.Brief.Overview
	if live := len(livePages(c)); shape.Pages != live {
		t.Errorf("the overview counts %d pages, and the model holds %d live ones", shape.Pages, live)
	}
	typed := 0
	for _, count := range shape.Types {
		typed += count.Pages
	}
	if typed != shape.Pages || len(shape.Types) < 3 {
		t.Errorf("the overview types %d of its %d pages in %d types: %+v", typed, shape.Pages, len(shape.Types), shape.Types)
	}
	if hub, found := hubAt(shape.Hubs, capabilityHub); !found || hub.Pages == 0 {
		t.Errorf("the overview's top of the tree is %+v, want %s with its pages", shape.Hubs, capabilityHub)
	}
	want := map[string]int{}
	inbox := findingsIn(t, c, "", finding.Open)
	for i := range inbox {
		want[inbox[i].Code]++
	}
	got := map[string]int{}
	for _, count := range shape.Findings {
		got[string(count.Code)] = count.Count
	}
	if counted(got) != counted(want) || len(want) == 0 {
		t.Errorf("the overview counts the findings by code as %s, and the inbox holds %s", counted(got), counted(want))
	}
	if overview.Brief.Links.Audited == 0 || overview.Brief.Links.Targets == 0 {
		t.Errorf("the overview's link health is %+v, want the audited pages and their targets", overview.Brief.Links)
	}
	if entities := overview.Totals.Entities; entities.Total != len(c.seeds) || entities.WithCanonicalPage != len(c.seeds) {
		t.Errorf("the overview's entities are %+v, want the %d seeded, each with its page", entities, len(c.seeds))
	}

	listed := ask[tools.SitesResponse](t, c, conversation, "sites", map[string]any{"view": "list"})
	if listed.Sites == nil || !slices.ContainsFunc(listed.Sites.Items, func(held sites.Site) bool { return held.ID == c.siteID }) {
		t.Errorf("sites list answered %+v, want this site", listed.Sites)
	}
	one := ask[tools.SitesResponse](t, c, conversation, "sites", map[string]any{"view": "get"})
	if one.Site == nil || one.Site.ID != c.siteID || !one.Site.Plugin.Installed {
		t.Errorf("sites get answered %+v, want this site with its plugin", one.Site)
	}
	t.Logf("the overview: %d pages in %d types, %d open findings in %d codes, link health %+v",
		shape.Pages, len(shape.Types), len(findingsIn(t, c, "", finding.Open)), len(want), overview.Brief.Links)
}

func treeOf(t *testing.T, c *capability, conversation string, args map[string]any) sitemodel.TreeAnswer {
	t.Helper()
	return ask[sitemodel.TreeAnswer](t, c, conversation, "pages_tree", args)
}

func pathsOf(nodes []contextpack.MapNode) []string {
	out := make([]string, 0, len(nodes))
	for i := range nodes {
		out = append(out, nodes[i].Page.Path)
	}
	slices.Sort(out)
	return out
}

func nodeAt(nodes []contextpack.MapNode, path string) (contextpack.MapNode, bool) {
	at := slices.IndexFunc(nodes, func(node contextpack.MapNode) bool { return node.Page.Path == path })
	if at < 0 {
		return contextpack.MapNode{}, false
	}
	return nodes[at], true
}

func assertEveryParentOrWhy(t *testing.T, view string, nodes []contextpack.MapNode) {
	t.Helper()

	for i := range nodes {
		if node := &nodes[i]; node.ParentID == "" && (node.Why == nil || node.Why.Reason == "") {
			t.Errorf("%s answers %s with no parent and no reason", view, node.Page.Path)
		}
	}
}

func readsThePageTree(t *testing.T, c *capability) {
	conversation := c.conversation(t)
	hub, category, bpc, tb := c.page(t, capabilityHub), c.page(t, capabilityCategory), c.page(t, capabilityBPC), c.page(t, capabilityTB)

	roots := treeOf(t, c, conversation, map[string]any{"view": "roots", "limit": 200})
	front, found := nodeAt(roots.Nodes.Items, capabilityFront)
	if !found || front.ParentID != "" || front.Why == nil || front.Why.Reason != contextpack.WhyRoot || front.Children == 0 {
		t.Fatalf("the roots are %v, want the front page with no parent because it is the root", pathsOf(roots.Nodes.Items))
	}
	assertEveryParentOrWhy(t, "roots", roots.Nodes.Items)

	children := treeOf(t, c, conversation, map[string]any{"view": "children", "pageId": front.Page.PageID, "limit": 200})
	top, found := nodeAt(children.Nodes.Items, capabilityHub)
	if !found || top.ParentID != front.Page.PageID || top.Depth != front.Depth+1 || len(children.Nodes.Items) != front.Children {
		t.Errorf("the children of the front page are %v (%d, the front page says %d), want %s one level down",
			pathsOf(children.Nodes.Items), len(children.Nodes.Items), front.Children, capabilityHub)
	}

	subtree := treeOf(t, c, conversation, map[string]any{"view": "subtree", "pageId": hub.ID, "depth": 2, "limit": 200})
	if subtree.Node == nil || subtree.Node.Page.Path != capabilityHub {
		t.Fatalf("the subtree view names %+v as its root, want %s", subtree.Node, capabilityHub)
	}
	for _, path := range []string{capabilityCategory, capabilityBPC, capabilityTB} {
		if _, held := nodeAt(subtree.Nodes.Items, path); !held {
			t.Errorf("the subtree of %s to depth 2 is %v, want %s in it", capabilityHub, pathsOf(subtree.Nodes.Items), path)
		}
	}
	for i := range subtree.Nodes.Items {
		if node := &subtree.Nodes.Items[i]; node.Depth <= subtree.Node.Depth || node.Depth > subtree.Node.Depth+2 {
			t.Errorf("the subtree to depth 2 holds %s at depth %d under a root at %d", node.Page.Path, node.Depth, subtree.Node.Depth)
		}
	}

	ancestors := treeOf(t, c, conversation, map[string]any{"view": "ancestors", "pageId": tb.ID})
	if got, want := pathsOf(ancestors.Nodes.Items), []string{capabilityFront, capabilityCategory, capabilityHub}; !slices.Equal(got, want) {
		t.Errorf("the ancestors of %s are %v, want %v", capabilityTB, got, want)
	}

	siblings := treeOf(t, c, conversation, map[string]any{"view": "siblings", "pageId": bpc.ID, "limit": 200})
	if _, held := nodeAt(siblings.Nodes.Items, capabilityTB); !held || slices.Contains(pathsOf(siblings.Nodes.Items), capabilityBPC) {
		t.Errorf("the siblings of %s are %v, want %s and not the page itself", capabilityBPC, pathsOf(siblings.Nodes.Items), capabilityTB)
	}
	paged := make([]contextpack.MapNode, 0, len(siblings.Nodes.Items))
	for cursor := ""; ; {
		args := map[string]any{"view": "siblings", "pageId": bpc.ID, "limit": 1}
		if cursor != "" {
			args["cursor"] = cursor
		}
		one := treeOf(t, c, conversation, args)
		paged = append(paged, one.Nodes.Items...)
		if one.Nodes.Next == "" {
			break
		}
		cursor = string(one.Nodes.Next)
	}
	if got, want := pathsOf(paged), pathsOf(siblings.Nodes.Items); !slices.Equal(got, want) || len(want) < 2 {
		t.Errorf("the siblings of %s paged one at a time are %v, in one call %v", capabilityBPC, got, want)
	}

	walk := treeOf(t, c, conversation, map[string]any{"view": "walk", "limit": 200})
	if walk.Nodes.Next != "" || walk.Generation == 0 || walk.Stale {
		t.Fatalf("the walk of %d pages answered next %q, generation %d, stale %v", len(walk.Nodes.Items), walk.Nodes.Next, walk.Generation, walk.Stale)
	}
	assertEveryParentOrWhy(t, "the walk", walk.Nodes.Items)
	reasons := map[contextpack.WhyReason]int{}
	for i := range walk.Nodes.Items {
		if node := &walk.Nodes.Items[i]; node.Why != nil {
			reasons[node.Why.Reason]++
			t.Logf("no parent for %s: %+v", node.Page.Path, *node.Why)
		}
	}

	categoryType := typeIDOf(t, c, category)
	typed := treeOf(t, c, conversation, map[string]any{"view": "subtree", "pageId": hub.ID, "depth": 32, "pageTypeId": categoryType, "limit": 200})
	if _, held := nodeAt(typed.Nodes.Items, capabilityCategory); !held {
		t.Errorf("the subtree of %s filtered to the category type is %v, want %s", capabilityHub, pathsOf(typed.Nodes.Items), capabilityCategory)
	}
	for i := range typed.Nodes.Items {
		if page := c.pages[typed.Nodes.Items[i].Page.Path]; page.PageTypeID == nil || *page.PageTypeID != categoryType {
			t.Errorf("the type filter let %s through, typed %v", page.Path, page.PageTypeID)
		}
	}
	t.Logf("the tree: %d roots, %d pages on the walk, reasons for no parent %v", len(roots.Nodes.Items), len(walk.Nodes.Items), reasons)
}

func typeIDOf(t *testing.T, c *capability, page pages.Page) string {
	t.Helper()

	if page.PageTypeID == nil {
		t.Fatalf("%s carries no page type", page.Path)
	}
	return *page.PageTypeID
}

func entityRead(t *testing.T, c *capability, conversation, entityID string) sitemodel.EntityDetail {
	t.Helper()

	answered := ask[tools.GraphReadResponse](t, c, conversation, "graph_read", map[string]any{"view": "entity", "entityId": entityID})
	if answered.Entity == nil || answered.Entity.ID != entityID {
		t.Fatalf("graph_read entity %s answered %+v", entityID, answered)
	}
	return *answered.Entity
}

func entityNames(rows []sitemodel.EntityRow) []string {
	out := make([]string, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].Name)
	}
	slices.Sort(out)
	return out
}

func readsTheEntityGraph(t *testing.T, c *capability) {
	conversation := c.conversation(t)
	healing := entityRead(t, c, conversation, c.entities["Healing Peptides"])
	if len(healing.Parents) != 1 || healing.Parents[0].EntityID != c.entities["Peptides"] || healing.Parents[0].Status != "approved" {
		t.Errorf("the parents of Healing Peptides are %+v, want Peptides, approved", healing.Parents)
	}
	if healing.Children != 2 || healing.ParentWhy != "" {
		t.Errorf("Healing Peptides holds %d children and says %q of its parent, want 2 and none", healing.Children, healing.ParentWhy)
	}
	if page := healing.CanonicalPage; page == nil || page.Path != capabilityCategory || page.ParentPath != capabilityHub {
		t.Errorf("the canonical page of Healing Peptides is %+v, want %s under %s", page, capabilityCategory, capabilityHub)
	}
	if healing.Tree == nil || healing.Tree.EntityID != c.entities["Peptides"] || healing.Tree.Path != capabilityHub {
		t.Errorf("the tree's parent of Healing Peptides is %+v, want Peptides at %s", healing.Tree, capabilityHub)
	}
	if healing.Definition != readDefinition || len(healing.Facts) != 1 {
		t.Errorf("the card of Healing Peptides is %q with %+v, want the definition and the fact written", healing.Definition, healing.Facts)
	}
	if !slices.ContainsFunc(healing.Keywords, func(held dto.Keyword) bool { return held.Text == "healing peptides" }) {
		t.Errorf("the keywords of Healing Peptides are %+v, want its own", healing.Keywords)
	}
	if !slices.Equal(healing.Anchors, []string{readAnchor}) {
		t.Errorf("the anchors of Healing Peptides are %v, want %q", healing.Anchors, readAnchor)
	}

	bpc := entityRead(t, c, conversation, c.entities["BPC-157"])
	if !slices.ContainsFunc(bpc.Related, func(held sitemodel.EntityLink) bool {
		return held.EntityID == c.entities["TB-500"] && held.Status == "approved"
	}) {
		t.Errorf("the related of BPC-157 are %+v, want TB-500, approved", bpc.Related)
	}

	under := ask[tools.GraphReadResponse](t, c, conversation, "graph_read", map[string]any{"view": "entities", "parentId": c.entities["Healing Peptides"]})
	if under.Entities == nil || !slices.Equal(entityNames(under.Entities.Items), []string{"BPC-157", "TB-500"}) {
		t.Errorf("the children of Healing Peptides are %+v, want BPC-157 and TB-500", under.Entities)
	}
	roots := ask[tools.GraphReadResponse](t, c, conversation, "graph_read", map[string]any{"view": "entities", "roots": true})
	if roots.Entities == nil || !slices.Equal(entityNames(roots.Entities.Items), []string{"Lab Guides", "Peptides"}) {
		t.Fatalf("the roots of the graph are %+v, want Lab Guides and Peptides", roots.Entities)
	}
	for _, row := range roots.Entities.Items {
		if row.Parent != nil || row.ParentWhy != sitemodel.NoParentEdge || row.CanonicalPage == nil {
			t.Errorf("the root %s reads %+v, want no parent with its reason and its page", row.Name, row)
		}
	}
	edges := ask[tools.GraphReadResponse](t, c, conversation, "graph_read", map[string]any{"view": "edges", "entityId": c.entities["BPC-157"]})
	kinds := map[string]int{}
	for i := range edges.Edges.Items {
		kinds[edges.Edges.Items[i].Kind]++
	}
	if kinds["parent"] != 1 || kinds["related"] != 1 {
		t.Errorf("the edges of BPC-157 are %+v, want its parent edge and its related edge", edges.Edges.Items)
	}
}

func readsBothHierarchiesAgree(t *testing.T, c *capability) {
	conversation := c.conversation(t)
	parentOf := map[string]string{}
	for _, seed := range c.seeds {
		parentOf[seed.name] = c.entities[seed.parent]
	}
	for _, seed := range c.seeds {
		detail := entityRead(t, c, conversation, c.entities[seed.name])
		graphParent := ""
		if len(detail.Parents) > 0 {
			graphParent = detail.Parents[0].EntityID
		}
		treeParent := ""
		if detail.Tree != nil {
			treeParent = detail.Tree.EntityID
		}
		if graphParent != parentOf[seed.name] || treeParent != parentOf[seed.name] {
			t.Errorf("%s reads its graph parent %q and its tree parent %q side by side, want both %q",
				seed.name, graphParent, treeParent, parentOf[seed.name])
		}
	}
	adoption := ask[tools.GraphReadResponse](t, c, conversation, "graph_read", map[string]any{"view": "adoption"})
	if adoption.Adoption == nil || adoption.Adoption.Adopting != 0 || adoption.Adoption.Hash == "" {
		t.Errorf("the adoption preview of a graph that follows the tree is %+v, want nothing to adopt", adoption.Adoption)
	}
	listed := ask[tools.FindingsResponse](t, c, conversation, "findings", map[string]any{"op": "list", "code": string(finding.EntityParentDiffers)})
	if listed.List == nil || len(listed.List.Items) != 0 {
		t.Errorf("a graph that follows the tree carries the entity_parent_differs findings %+v", listed.List)
	}
}

func readsAPageInFull(t *testing.T, c *capability) {
	conversation := c.conversation(t)
	page, parent := c.page(t, capabilityTB), c.page(t, capabilityCategory)
	answered := ask[tools.PagesGetResponse](t, c, conversation, "pages_get", map[string]any{"pageId": page.ID})
	model := answered.Page
	if model == nil || model.Page.PageID != page.ID || model.Type.Name == "" || model.Type.Reason == "" {
		t.Fatalf("pages_get answered %+v, want %s with its type and why", answered.Page, page.Path)
	}
	if model.Parent.ParentID != parent.ID || len(model.Parent.Trail) == 0 || model.Parent.Trail[0].PageID != parent.ID {
		t.Errorf("the parent of %s is %+v, want %s first on its trail, by id", page.Path, model.Parent, parent.Path)
	}
	for i := range model.Parent.Trail {
		if crumb := &model.Parent.Trail[i]; crumb.PageID == "" || crumb.Path == "" {
			t.Errorf("the trail of %s holds a crumb without its id: %+v", page.Path, *crumb)
		}
	}
	if answered.Template == nil || answered.Template.ID == "" || answered.Template.Source == "" {
		t.Errorf("the template a run would use on %s is %+v, want it with where it comes from", page.Path, answered.Template)
	}
	if !slices.ContainsFunc(model.Fields, func(held sitemodel.PageField) bool { return held.Key == "body" && held.Enabled && held.Owner != "" }) {
		t.Errorf("the fields of %s are %+v, want the body with its scope and its owner", page.Path, model.Fields)
	}
	want := make([]string, 0)
	inbox := findingsIn(t, c, "", finding.Open)
	for i := range inbox {
		if inbox[i].PageID == page.ID {
			want = append(want, inbox[i].ID)
		}
	}
	got := make([]string, 0, len(model.Findings))
	for _, held := range model.Findings {
		got = append(got, held.ID)
	}
	slices.Sort(want)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("the page model names the findings %v, and the inbox holds %v on %s", got, want, page.Path)
	}

	raw, err := wordpress(t, c.core, c.siteID).GetRaw(t.Context(), wp.ItemType(page.WPType), *page.WPID)
	if err != nil {
		t.Fatalf("read %s from the site: %v", page.Path, err)
	}
	body := block.Parse(raw.Content)
	if len(body.Blocks) < 2 || model.Body.Blocks != len(body.Blocks) {
		t.Fatalf("%s holds %d blocks on the site and the model counts %d; reset the stand", page.Path, len(body.Blocks), model.Body.Blocks)
	}
	read, hash, calls := readTheBodyWhole(t, c, conversation, page.ID)
	if hash != block.BodyHash(raw.Content) {
		t.Errorf("the body read has the hash %s, and the site's is %s", hash, block.BodyHash(raw.Content))
	}
	for _, held := range body.Blocks {
		if want := strings.Join(strings.Fields(held.Text()), " "); read[held.Index] != want {
			t.Errorf("block %d of %s reads %q through pages_blocks, and the site holds %q", held.Index, page.Path, read[held.Index], want)
		}
	}
	if len(read) != len(body.Blocks) || calls < len(body.Blocks) {
		t.Errorf("the paged read gave %d blocks in %d calls for the %d the site holds", len(read), calls, len(body.Blocks))
	}
	t.Logf("%s read whole: %d blocks in %d calls of one", page.Path, len(read), calls)
}

func readTheBodyWhole(t *testing.T, c *capability, conversation, pageID string) (texts map[int]string, hash string, calls int) {
	t.Helper()

	parts := map[int][]string{}
	cursor := ""
	for {
		args := map[string]any{"op": "list", "pageId": pageID, "text": true, "limit": 1}
		if cursor != "" {
			args["cursor"] = cursor
		}
		listed := ask[tools.PagesBlocksResponse](t, c, conversation, "pages_blocks", args)
		calls++
		if listed.Blocks == nil {
			t.Fatalf("pages_blocks list answered no blocks for %s", pageID)
		}
		hash = listed.Blocks.BodyHash
		rows := listed.Blocks.Blocks.Items
		for i := range rows {
			parts[rows[i].Index] = append(parts[rows[i].Index], rows[i].Text)
		}
		if listed.Blocks.Blocks.Next == "" {
			break
		}
		cursor = string(listed.Blocks.Blocks.Next)
	}
	texts = make(map[int]string, len(parts))
	for index, held := range parts {
		texts[index] = strings.Join(held, "")
	}
	return texts, hash, calls
}

func findPages(t *testing.T, c *capability, conversation string, args map[string]any) []contextpack.MapNode {
	t.Helper()

	args["limit"] = 200
	listed := ask[paging.List[contextpack.MapNode]](t, c, conversation, "pages_find", args)
	if listed.Next != "" {
		t.Fatalf("pages_find %v did not fit one page of 200", args)
	}
	return listed.Items
}

func readsManyPagesAtOnce(t *testing.T, c *capability) {
	conversation := c.conversation(t)
	hub, bpc, tb, recon := c.page(t, capabilityHub), c.page(t, capabilityBPC), c.page(t, capabilityTB), c.page(t, capabilityRecon)

	byIDs := findPages(t, c, conversation, map[string]any{"pageIds": []string{bpc.ID, tb.ID, recon.ID}})
	if got := pathsOf(byIDs); !slices.Equal(got, []string{capabilityBPC, capabilityRecon, capabilityTB}) {
		t.Errorf("pages_find by ids answered %v", got)
	}
	for i := range byIDs {
		if node := &byIDs[i]; node.Page.Type == "" || node.ParentID == "" || node.ParentReason == "" {
			t.Errorf("pages_find answered %s without its key facts: %+v", node.Page.Path, *node)
		}
	}

	categoryType := typeIDOf(t, c, c.page(t, capabilityCategory))
	want := make([]string, 0)
	live := livePages(c)
	for path := range live {
		if typed := live[path].PageTypeID; typed != nil && *typed == categoryType {
			want = append(want, path)
		}
	}
	slices.Sort(want)
	if got := pathsOf(findPages(t, c, conversation, map[string]any{"pageTypeId": categoryType})); !slices.Equal(got, want) {
		t.Errorf("pages_find by the category type answered %v, and the model types %v so", got, want)
	}

	under := findPages(t, c, conversation, map[string]any{"under": hub.ID})
	subtree := treeOf(t, c, conversation, map[string]any{"view": "subtree", "pageId": hub.ID, "depth": 32, "limit": 200})
	if got, tree := pathsOf(under), pathsOf(subtree.Nodes.Items); !slices.Equal(got, tree) {
		t.Errorf("pages_find under %s answered %v, and the tree holds %v below it", capabilityHub, got, tree)
	}

	mapped := findPages(t, c, conversation, map[string]any{"entityId": c.entities["TB-500"]})
	if got := pathsOf(mapped); !slices.Equal(got, []string{capabilityTB}) {
		t.Errorf("pages_find by the entity TB-500 answered %v", got)
	}
	assertEveryParentOrWhy(t, "pages_find", byIDs)
}

func sectionIDs(t *testing.T, c *capability, conversation string, root pages.Page) []string {
	t.Helper()

	subtree := treeOf(t, c, conversation, map[string]any{"view": "subtree", "pageId": root.ID, "depth": 32, "limit": 200})
	if subtree.Node == nil {
		t.Fatalf("the subtree of %s names no root", root.Path)
	}
	nodes := append([]contextpack.MapNode{*subtree.Node}, subtree.Nodes.Items...)
	ids := make([]string, 0, 2*len(nodes))
	for i := range nodes {
		ids = append(ids, nodes[i].Page.PageID)
		if nodes[i].Twin != nil {
			ids = append(ids, nodes[i].Twin.PageID)
		}
	}
	return ids
}

func linksOfSection(t *testing.T, c *capability, conversation string, args map[string]any) sitemodel.LinksAnswer {
	t.Helper()

	args["limit"] = 200
	return ask[sitemodel.LinksAnswer](t, c, conversation, "links", args)
}

func readsTheLinksOfASection(t *testing.T, c *capability) {
	conversation := c.conversation(t)
	hub := c.page(t, capabilityHub)
	ids := sectionIDs(t, c, conversation, hub)
	audit, err := c.core.Reports.LinkAuditPages(t.Context(), reports.LinkAuditPagesRequest{SiteID: c.siteID, PageIDs: ids})
	if err != nil {
		t.Fatalf("read the link audit of the section: %v", err)
	}
	want := sitemodel.SectionTotals{}
	owed, missing := 0, 0
	for i := range audit.Pages {
		row := audit.Pages[i].Page
		want.Pages++
		want.Required += row.Required
		want.Satisfied += row.Satisfied
		want.MissingRequired += row.MissingRequired
		want.OffGraph += row.OffGraph
		if row.Orphan {
			want.Orphans++
		}
		owed += len(audit.Pages[i].Required)
		for j := range audit.Pages[i].Required {
			if audit.Pages[i].Required[j].State == string(reports.LinkMissing) {
				missing++
			}
		}
	}

	page := linksOfSection(t, c, conversation, map[string]any{"view": "page", "pageId": hub.ID})
	if page.Page == nil || page.Page.Page.Path != capabilityHub || page.Inbound == nil ||
		!slices.ContainsFunc(page.Page.Required, func(held reports.RequiredLink) bool { return held.TargetPath == capabilityCategory }) {
		t.Errorf("links page of %s answered %+v, want its links in and out with what it owes %s", capabilityHub, page.Page, capabilityCategory)
	}

	section := linksOfSection(t, c, conversation, map[string]any{"view": "section", "pageId": hub.ID})
	if section.Root == nil || section.Root.Path != capabilityHub || section.Totals == nil || *section.Totals != want ||
		len(section.Pages.Items) != want.Pages {
		listed := make([]string, 0, len(section.Pages.Items))
		for i := range section.Pages.Items {
			listed = append(listed, section.Pages.Items[i].Path)
		}
		audited := make([]string, 0, len(audit.Pages))
		for i := range audit.Pages {
			audited = append(audited, audit.Pages[i].Page.Path)
		}
		slices.Sort(listed)
		slices.Sort(audited)
		t.Errorf("links section of %s answered root %+v totals %+v over the pages %v, the audit of its tree and twins says %+v over %v",
			capabilityHub, section.Root, section.Totals, listed, want, audited)
	}

	inside := map[string]bool{}
	for _, id := range ids {
		inside[id] = true
	}
	linked := linksOfSection(t, c, conversation, map[string]any{"view": "map", "pageId": hub.ID})
	for _, held := range linked.Links.Items {
		if !inside[held.FromPageID] || held.Inside != inside[held.ToPageID] || held.Anchor == "" && held.Ours {
			t.Errorf("the section's link map holds %+v, which is not a link from the section as it says", held)
		}
	}

	all := linksOfSection(t, c, conversation, map[string]any{"view": "owed", "pageId": hub.ID})
	gaps := linksOfSection(t, c, conversation, map[string]any{"view": "owed", "pageId": hub.ID, "missing": true})
	if len(all.Owed.Items) != owed || len(gaps.Owed.Items) != missing {
		t.Errorf("links owed answered %d links and %d missing, the audit owes %d and misses %d", len(all.Owed.Items),
			len(gaps.Owed.Items), owed, missing)
	}
	paged := make([]sitemodel.OwedLink, 0, owed)
	for cursor := ""; ; {
		args := map[string]any{"view": "owed", "pageId": hub.ID, "limit": 2}
		if cursor != "" {
			args["cursor"] = cursor
		}
		one := ask[sitemodel.LinksAnswer](t, c, conversation, "links", args)
		paged = append(paged, one.Owed.Items...)
		if one.Owed.Next == "" {
			break
		}
		cursor = string(one.Owed.Next)
	}
	if len(paged) != owed || !slices.EqualFunc(paged, all.Owed.Items, func(a, b sitemodel.OwedLink) bool {
		return a.FromPageID == b.FromPageID && a.TargetEntityID == b.TargetEntityID && a.Relation == b.Relation
	}) {
		t.Errorf("links owed paged two at a time gave %d links, in one call %d", len(paged), len(all.Owed.Items))
	}

	orphans := linksOfSection(t, c, conversation, map[string]any{"view": "orphans", "pageId": hub.ID})
	if len(orphans.Pages.Items) != want.Orphans {
		t.Errorf("links orphans answered %d pages, the audit counts %d", len(orphans.Pages.Items), want.Orphans)
	}
	for i := range orphans.Pages.Items {
		if row := &orphans.Pages.Items[i]; !row.Orphan {
			t.Errorf("links orphans answered %s, which has inbound links", row.Path)
		}
	}

	suggested := linksOfSection(t, c, conversation, map[string]any{"view": "suggest", "pageId": hub.ID})
	links := 0
	for i := range suggested.Suggestions.Items {
		held := &suggested.Suggestions.Items[i]
		if held.Kind == sitemodel.SuggestParentEdge {
			t.Errorf("a graph that follows the tree is suggested the parent edge %+v", *held)
		}
		if held.Kind == sitemodel.SuggestLink {
			links++
		}
	}
	if links != missing {
		t.Errorf("links suggest answered %d links to add, the audit misses %d", links, missing)
	}
	t.Logf("the section %s: %+v, %d links on its map, %d owed, %d missing, %d orphans",
		capabilityHub, want, len(linked.Links.Items), owed, missing, want.Orphans)
}

func searchesByWords(t *testing.T, c *capability) {
	conversation := c.conversation(t)
	if got := pathsOf(findPages(t, c, conversation, map[string]any{"query": "reconstitution"})); !slices.Contains(got, capabilityRecon) {
		t.Errorf("pages_find for the words reconstitution answered %v", got)
	}
	healing := pathsOf(findPages(t, c, conversation, map[string]any{"query": "healing"}))
	if !slices.Contains(healing, capabilityCategory) || slices.Contains(healing, capabilityGuides) {
		t.Errorf("pages_find for the word healing answered %v", healing)
	}
	sample := c.page(t, "/sample-page/")
	own := []dto.Keyword{{Text: "capability sample keyword"}}
	if _, err := c.core.Pages.Update(t.Context(), pages.UpdateRequest{ID: sample.ID, Keywords: &own}); err != nil {
		t.Fatalf("give %s a keyword of its own: %v", sample.Path, err)
	}
	if byOwn := pathsOf(findPages(t, c, conversation, map[string]any{"query": "sample keyword"})); !slices.Equal(byOwn, []string{sample.Path}) {
		t.Errorf("pages_find for a keyword the page holds answered %v, want %s", byOwn, sample.Path)
	}
	recon := c.page(t, capabilityRecon)
	if len(recon.Keywords) != 0 {
		t.Fatalf("%s holds keywords of its own, %+v, so it proves nothing about following its entity", recon.Path, recon.Keywords)
	}
	byEntity := pathsOf(findPages(t, c, conversation, map[string]any{"query": "peptide reconstitution"}))
	if !slices.Contains(byEntity, capabilityRecon) {
		t.Errorf("pages_find for peptide reconstitution, the keyword %s takes from its entity, answered %v", capabilityRecon, byEntity)
	}
	named := ask[tools.GraphReadResponse](t, c, conversation, "graph_read", map[string]any{"view": "entities", "query": "bpc"})
	if named.Entities == nil || !slices.Equal(entityNames(named.Entities.Items), []string{"BPC-157"}) {
		t.Errorf("graph_read entities by the name bpc answered %+v", named.Entities)
	}
}

func readsFindingsRunsCostsAndImports(t *testing.T, c *capability) {
	if _, err := c.core.Imports.SaveMapping(t.Context(), imports.SaveMappingRequest{Mapping: imports.Mapping{
		SiteID: c.siteID, Name: readMapping, Columns: map[string]string{"path": "URL", "title": "Title", "entity": "Entity"},
	}}); err != nil {
		t.Fatalf("save a mapping as the person: %v", err)
	}
	opened := len(c.script.Requests())
	conversation := c.conversation(t)

	want := idsOf(findingsIn(t, c, "", finding.Open))
	got := make([]string, 0, len(want))
	cursor, calls := "", 0
	for {
		args := map[string]any{"op": "list", "limit": 2}
		if cursor != "" {
			args["cursor"] = cursor
		}
		listed := ask[tools.FindingsResponse](t, c, conversation, "findings", args)
		calls++
		got = append(got, idsOf(listed.List.Items)...)
		if listed.List.Next == "" {
			break
		}
		cursor = string(listed.List.Next)
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) || calls < 2 {
		t.Errorf("findings list by cursor gave %d findings in %d calls, the inbox holds %d", len(got), calls, len(want))
	}
	code := findingsIn(t, c, "", finding.Open)[0].Code
	byCode := ask[tools.FindingsResponse](t, c, conversation, "findings", map[string]any{"op": "list", "code": code, "limit": 200})
	for i := range byCode.List.Items {
		if held := byCode.List.Items[i].Code; held != code {
			t.Errorf("findings list of the code %s answered a %s", code, held)
		}
	}

	listed := ask[tools.RunsResponse](t, c, conversation, "runs", map[string]any{"view": "list"})
	if listed.Runs == nil || len(listed.Runs.Items) == 0 {
		t.Fatalf("runs list answered %+v, want the sync that observed the site", listed.Runs)
	}
	synced := listed.Runs.Items[0]
	one := ask[tools.RunsResponse](t, c, conversation, "runs", map[string]any{"view": "get", "runId": synced.ID})
	if one.Run == nil || one.Run.Kind != "sync" || one.Run.Status != "completed" {
		t.Errorf("runs get answered %+v, want the completed sync", one.Run)
	}
	items := ask[tools.RunsResponse](t, c, conversation, "runs", map[string]any{"view": "items", "runId": synced.ID})
	if items.Items == nil || len(items.Items.Items) == 0 || items.Items.Items[0].Status != "completed" {
		t.Errorf("runs items answered %+v, want the sync's completed item", items.Items)
	}

	asked := len(c.script.Requests()) - opened
	spent := ask[tools.ModelsResponse](t, c, conversation, "models", map[string]any{"view": "usage", "conversationId": conversation})
	direct, err := c.core.Models.UsageSummary(t.Context(), models.UsageSummaryRequest{ConversationID: conversation})
	if err != nil {
		t.Fatalf("read the conversation's spend: %v", err)
	}
	if spent.Usage == nil || spent.Usage.Calls != asked+1 || direct.Calls != asked+2 || spent.Usage.Usage.Total == 0 {
		t.Errorf("models usage of the conversation answered %+v after %d calls to the model, the ledger then holds %+v",
			spent.Usage, asked+1, direct)
	}

	mappings := ask[imports.ListMappingsResponse](t, c, conversation, "imports", map[string]any{"op": "mappings"})
	if !slices.ContainsFunc(mappings.Mappings, func(held imports.Mapping) bool {
		return held.Name == readMapping && held.Columns["path"] == "URL"
	}) {
		t.Errorf("imports mappings answered %+v, want the saved mapping", mappings.Mappings)
	}
}

func readsTypesScopesTemplatesAndRules(t *testing.T, c *capability) {
	conversation := c.conversation(t)
	listed := ask[tools.PageTypesResponse](t, c, conversation, "pagetypes", map[string]any{"op": "list"})
	categoryType := typeIDOf(t, c, c.page(t, capabilityCategory))
	counts := map[string]int{}
	live := livePages(c)
	for path := range live {
		if typed := live[path].PageTypeID; typed != nil {
			counts[*typed]++
		}
	}
	at := slices.IndexFunc(listed.Types, func(held tools.PageTypeBrief) bool { return held.ID == categoryType })
	if at < 0 || listed.Types[at].Pages != counts[categoryType] || !slices.Contains(listed.Types[at].Writes, "body") {
		t.Fatalf("pagetypes list answered %+v, want the category type with its %d pages writing the body", listed.Types, counts[categoryType])
	}
	whole := ask[tools.PageTypesResponse](t, c, conversation, "pagetypes", map[string]any{"op": "get", "pageTypeId": categoryType})
	if whole.Type == nil || whole.Type.ID != categoryType || len(whole.Fields) == 0 ||
		!slices.ContainsFunc(whole.Type.Scopes, func(held pagetypes.Scope) bool { return held.Field == "body" && held.Enabled }) {
		t.Errorf("pagetypes get answered %+v with %d catalog fields, want the type whole with its body scope", whole.Type, len(whole.Fields))
	}

	shipped := ask[paging.List[templates.Template]](t, c, conversation, "templates", map[string]any{"op": "list", "limit": 200})
	names := make([]string, 0, len(shipped.Items))
	for i := range shipped.Items {
		names = append(names, shipped.Items[i].Name)
	}
	if !slices.Contains(names, "Hub") || !slices.Contains(names, "Guide") {
		t.Errorf("templates list answered %v, want the shipped Hub and Guide", names)
	}

	guides := c.page(t, capabilityGuides)
	rules := ask[tools.PagetypesClassifyResponse](t, c, conversation, "pagetypes_classify", map[string]any{"op": "rules"})
	if rules.Rules == nil || rules.Rules.Fallback == nil || len(rules.Rules.Rules) < 5 ||
		!slices.ContainsFunc(rules.Rules.Rules, func(held pagetypes.Rule) bool { return held.ID == guides.TypeRuleID }) {
		t.Fatalf("pagetypes_classify rules answered %+v, want the ordered rules, the fallback and the one that typed %s", rules.Rules, capabilityGuides)
	}
	tested := ask[tools.PagetypesClassifyResponse](t, c, conversation, "pagetypes_classify", map[string]any{
		"op": "test_rule", "ruleId": guides.TypeRuleID, "pageId": guides.ID,
	})
	if tested.Tested == nil || tested.Tested.Verdict != "met" || tested.Tested.Facts.Path != capabilityGuides {
		t.Errorf("pagetypes_classify test_rule answered %+v, want the rule met on %s", tested.Tested, capabilityGuides)
	}
	preview := ask[tools.PagetypesClassifyResponse](t, c, conversation, "pagetypes_classify", map[string]any{"op": "preview"})
	if preview.Hash == "" || preview.Moving != 0 {
		t.Errorf("right after a sync the classification preview moves %d pages: %+v", preview.Moving, preview.Groups)
	}
}
