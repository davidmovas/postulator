//go:build e2e

package e2e_test

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/graph"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	writerInstruction = "Open with what the pages below this one share, then send the reader to each of them."
	ourText           = "The shop checks every claim on this page about"
	keptRule          = "The current value is the client's: keep it as it stands and write only what it lacks."
	plannedState      = "WordPress type: page; planned, not on the site yet"
	noLinks           = "Links: none in this field"

	headingPicture  = "THE FULL PICTURE"
	headingPage     = "THE PAGE"
	headingRules    = "THE FIELD'S RULES"
	headingTrail    = "THE TRAIL, NEAREST PARENT FIRST"
	headingTargets  = "PAGES THIS FIELD LINKS TO"
	headingFields   = "ITS OTHER FIELDS"
	headingKeywords = "KEYWORDS, BY VOLUME"
	headingListed   = "PRODUCTS IT LISTS"
	headingTwin     = "PRODUCTS OF ITS STORE CATEGORY"
)

type seededEntity struct {
	name    string
	path    string
	parent  string
	keyword string
	volume  int
}

type plannedChild struct {
	path    string
	name    string
	keyword string
}

type clientEditPlan struct {
	path string
	from string
	to   string
}

type writerStand struct {
	existing   string
	clientPage string
	preset     pagetype.Preset
	empty      string
	lists      bool
	child      plannedChild
	kept       string
	keptText   string
	edit       clientEditPlan
	graph      []seededEntity

	product         string
	productCategory string
	productHub      string
}

func writerStandOf(t *testing.T, env environment) writerStand {
	t.Helper()

	switch env.stand {
	case "vanilla":
		return writerStand{
			existing:   "/guide/setup/",
			clientPage: "/guide/setup/",
			preset:     pagetype.Guide,
			empty:      "/guide/setup/grinding/",
			child:      plannedChild{path: "/guide/setup/grinding/", name: "Grinding", keyword: "grinding coffee beans"},
			edit: clientEditPlan{
				path: "/guide/", from: "what you bought.", to: "what you bought, and how long each bag keeps.",
			},
			graph: []seededEntity{
				{name: "Guide", path: "/guide/", keyword: "coffee guide", volume: 880},
				{name: "Setup", path: "/guide/setup/", parent: "Guide", keyword: "coffee setup", volume: 320},
			},
		}
	case "replica":
		return writerStand{
			existing:   "/peptides/",
			clientPage: "/tb-500/",
			preset:     pagetype.Landing,
			empty:      "/healing-peptides/",
			lists:      true,
			child:      plannedChild{path: "/peptides/peptide-storage/", name: "Peptide Storage", keyword: "peptide storage"},
			kept:       "/lab-guides/",
			keptText:   "Step-by-step guides for handling research peptides in the laboratory.",
			edit: clientEditPlan{
				path: "/reconstitution/", from: "Swirl gently and never shake the vial.",
				to: "Swirl gently, never shake the vial and keep it cold.",
			},
			graph: []seededEntity{
				{name: "Peptides", path: "/peptides/", keyword: "research peptides", volume: 12000},
				{name: "Healing Peptides", path: "/healing-peptides/", parent: "Peptides", keyword: "healing peptides", volume: 2400},
				{name: "BPC-157", path: "/bpc-157/", parent: "Healing Peptides", keyword: "bpc-157", volume: 9000},
				{name: "TB-500", path: "/tb-500/", parent: "Healing Peptides", keyword: "tb-500", volume: 6000},
				{name: "Lab Guides", path: "/lab-guides/", keyword: "peptide lab guides", volume: 300},
				{name: "Reconstitution", path: "/reconstitution/", parent: "Lab Guides", keyword: "peptide reconstitution", volume: 1300},
			},
			product:         "/product/bpc-157-peptide/",
			productCategory: "healing-peptides",
			productHub:      "/healing-peptides/",
		}
	}
	t.Fatalf("E2E_STAND is %q, and the writer test knows the vanilla and the replica stands", env.stand)
	return writerStand{}
}

func TestTheWriterSeesTheFullPicture(t *testing.T) {
	live := newSite(t)
	requirePlugin(t, live.env, true)
	stand := writerStandOf(t, live.env)

	edit := &clientEdit{env: live.env, root: repoRoot(t)}
	script := fake.NewScripted(
		fake.Reply{Step: steps.NameGenerateBody, Match: "Path: " + stand.edit.path + "\n", Make: edit.reply},
		fake.Reply{Step: steps.NameGenerateBody, Make: bodyFromPrompt},
	)
	core := openCoreWith(t, script)
	owner, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name: "Written Shop", BaseURL: live.env.baseURL, Username: live.env.user, Password: live.env.pass, AllowInsecure: true,
	})
	if err != nil {
		t.Fatalf("create the site: %v", err)
	}
	siteID := owner.Site.ID
	assertTheTypesCarryTheShippedTemplates(t, core, siteID)
	runSync(t, core, siteID)

	stored := pagesByPath(t, core.Pages, siteID)
	entities := seedStandGraph(t, core, siteID, stand.graph)
	existing := stored[stand.existing]
	child := planChild(t, core, siteID, existing, entities[entityAt(stand.graph, stand.existing)], stand.child)
	models := modelsOf(t, core, siteID)
	kind := instructTheBody(t, core, siteID, models[stand.existing].Page)
	if kind.Preset != stand.preset {
		t.Fatalf("%s is typed %s (%s), want the stand's %s type", stand.existing, kind.Name, kind.Preset, stand.preset)
	}

	written := startRun(t, core, siteID, run.PublishDraft, writing(), child.ID, existing.ID)
	awaitRun(t, core.Runs, written)

	byID := make(map[string]string, len(models))
	for path := range models {
		byID[models[path].Page.ID] = path
	}
	parentOfExisting := parentIn(t, models, byID, stand.existing)
	parentOfEmpty := parentIn(t, models, byID, stand.empty)
	assertThePicture(t, bodyPromptOf(t, script, stand.existing), pictureWant{
		path: stand.existing, parentPath: parentOfExisting, parentTitle: shownTitle(models[parentOfExisting]),
		instruction: writerInstruction, titleLine: titleLineOf(t, models[stand.existing]),
		products: productTitles(models, stand.lists), target: stand.empty, targetUnder: shownTitle(models[parentOfEmpty]),
	})
	assertThePicture(t, bodyPromptOf(t, script, stand.child.path), pictureWant{
		path: stand.child.path, planned: true, parentPath: stand.existing, parentTitle: shownTitle(models[stand.existing]),
	})

	client := wordpress(t, core, siteID)
	if stand.kept != "" {
		assertTheClientsTextIsKept(t, core, script, client, siteID, stored[stand.kept], stand.keptText)
	}
	assertAClientEditPausesThePublish(t, core, client, edit, siteID, stored[stand.edit.path], stand.edit)
}

func writing(publish ...string) []template.StepSpec {
	recipe := make([]template.StepSpec, 0, 2+len(publish))
	recipe = append(recipe,
		template.StepSpec{Name: steps.NameResolveContext, Enabled: true},
		template.StepSpec{Name: steps.NameGenerateBody, Enabled: true},
	)
	for _, name := range publish {
		recipe = append(recipe, template.StepSpec{Name: name, Enabled: true})
	}
	return recipe
}

func startRun(t *testing.T, core *app.Core, siteID string, mode run.PublishMode, recipe []template.StepSpec,
	pageIDs ...string) string {
	t.Helper()

	started, err := core.Runs.Start(t.Context(), runs.StartRequest{
		SiteID: siteID, PageIDs: pageIDs, PublishMode: string(mode), Recipe: recipe,
	})
	if err != nil {
		t.Fatalf("start the run over %v: %v", pageIDs, err)
	}
	if len(started.Added) != 0 {
		t.Fatalf("the run pulled in %+v, want only the pages it was started over", started.Added)
	}
	return started.RunID
}

func seedStandGraph(t *testing.T, core *app.Core, siteID string, seeds []seededEntity) map[string]string {
	t.Helper()

	stored := pagesByPath(t, core.Pages, siteID)
	ids := make(map[string]string, len(seeds))
	for i := range seeds {
		seed := &seeds[i]
		page, held := stored[seed.path]
		if !held || page.WPID == nil {
			t.Fatalf("the stand holds no %s on the site to map %s to", seed.path, seed.name)
		}
		volume := seed.volume
		created, err := core.Graph.CreateEntity(t.Context(), graph.CreateEntityRequest{
			SiteID: siteID, Name: seed.name, Keywords: []dto.Keyword{{Text: seed.keyword, Volume: &volume}}, ParentID: ids[seed.parent],
		})
		if err != nil {
			t.Fatalf("create the entity %s: %v", seed.name, err)
		}
		ids[seed.name] = created.Entity.ID
		if _, err = core.Pages.MapToEntity(t.Context(), pages.MapToEntityRequest{PageID: page.ID, EntityID: created.Entity.ID}); err != nil {
			t.Fatalf("map %s to %s: %v", seed.path, seed.name, err)
		}
		if _, err = core.Pages.SetCanonical(t.Context(), pages.SetCanonicalRequest{EntityID: created.Entity.ID, PageID: page.ID}); err != nil {
			t.Fatalf("make %s the page of %s: %v", seed.path, seed.name, err)
		}
	}
	return ids
}

func entityAt(seeds []seededEntity, path string) string {
	for i := range seeds {
		if seeds[i].path == path {
			return seeds[i].name
		}
	}
	return ""
}

func planChild(t *testing.T, core *app.Core, siteID string, parent pages.Page, parentEntityID string, child plannedChild) pages.Page {
	t.Helper()

	if parentEntityID == "" {
		t.Fatalf("the stand's graph maps nothing to %s, the parent of %s", parent.Path, child.path)
	}
	created, err := core.Graph.CreateEntity(t.Context(), graph.CreateEntityRequest{
		SiteID: siteID, Name: child.name, Keywords: []dto.Keyword{{Text: child.keyword}}, ParentID: parentEntityID,
	})
	if err != nil {
		t.Fatalf("create the entity %s: %v", child.name, err)
	}
	resolved, err := core.Templates.ResolveForPage(t.Context(), templates.ResolveForPageRequest{PageID: parent.ID})
	if err != nil {
		t.Fatalf("resolve the template of %s: %v", parent.Path, err)
	}
	entityID, templateID := created.Entity.ID, resolved.TemplateID
	planted, err := core.Pages.Create(t.Context(), pages.CreateRequest{
		SiteID: siteID, Path: child.path, Title: child.name, EntityID: &entityID, TemplateID: &templateID,
	})
	if err != nil {
		t.Fatalf("plan %s: %v", child.path, err)
	}
	if _, err = core.Pages.SetCanonical(t.Context(), pages.SetCanonicalRequest{EntityID: entityID, PageID: planted.Page.ID}); err != nil {
		t.Fatalf("make %s the page of %s: %v", child.path, child.name, err)
	}
	if planted.Page.ParentPageID == nil || *planted.Page.ParentPageID != parent.ID {
		t.Fatalf("%s sits under %v, want %s", child.path, planted.Page.ParentPageID, parent.Path)
	}
	return planted.Page
}

func instructTheBody(t *testing.T, core *app.Core, siteID string, page pagemap.Page) pagetype.Type {
	t.Helper()

	set, err := core.PageTypes.Set(t.Context(), siteID)
	if err != nil {
		t.Fatalf("load the site's types: %v", err)
	}
	kind := set.TypeOf(page)
	scope, _ := set.Scope(kind.ID, field.Body)
	if !scope.Links.Links {
		t.Fatalf("the %s type links nothing from its body, so the picture names no page to link to", kind.Name)
	}
	if _, err = core.PageTypes.SetScope(t.Context(), pagetypes.SetScopeRequest{
		PageTypeID: kind.ID, Field: string(field.Body), Enabled: true, AIWrites: true, Instruction: writerInstruction,
		MinLength: scope.MinLength, MaxLength: scope.MaxLength, Rewrites: scope.Rewrites, Links: scope.Links,
		Label: scope.Label, Provider: string(scope.Provider),
	}); err != nil {
		t.Fatalf("give the %s type's body an instruction: %v", kind.Name, err)
	}
	return kind
}

func parentIn(t *testing.T, models map[string]observe.PageModel, byID map[string]string, path string) string {
	t.Helper()

	model, held := models[path]
	if !held || model.Page.ParentPageID == nil {
		t.Fatalf("the model holds no parent for %s", path)
	}
	parent, known := byID[*model.Page.ParentPageID]
	if !known {
		t.Fatalf("the parent of %s is not a page of the model", path)
	}
	return parent
}

func shownTitle(model observe.PageModel) string {
	shown := map[field.Key]string{}
	for i := range model.Fields {
		shown[model.Fields[i].Key] = model.Fields[i].Current
	}
	for _, held := range []string{shown[field.Title], model.Page.Title, shown[field.H1], model.Page.H1} {
		if held != "" {
			return held
		}
	}
	return model.Page.Path
}

func titleLineOf(t *testing.T, model observe.PageModel) string {
	t.Helper()

	for i := range model.Fields {
		if model.Fields[i].Key == field.Title && strings.TrimSpace(model.Fields[i].Current) != "" {
			return "- Title: " + strings.Join(strings.Fields(model.Fields[i].Current), " ")
		}
	}
	t.Fatalf("%s carries no title field", model.Page.Path)
	return ""
}

func productTitles(models map[string]observe.PageModel, lists bool) []string {
	out := make([]string, 0)
	if !lists {
		return out
	}
	for path := range models {
		if page := models[path].Page; page.WPType == pagemap.WPProduct && page.Status != pagemap.StatusArchived {
			out = append(out, shownTitle(models[path]))
		}
	}
	slices.Sort(out)
	return out
}

func userPrompt(req port.Request) string {
	for i := len(req.Messages) - 1; i >= 0; i-- {
		message := &req.Messages[i]
		if message.Role == port.RoleUser && !strings.HasPrefix(message.Text, fake.JSONDirective) {
			return message.Text
		}
	}
	return ""
}

func bodyPromptOf(t *testing.T, script *fake.Scripted, path string) string {
	t.Helper()

	requests := script.Requests()
	prompt := ""
	for i := range requests {
		if requests[i].Meta.Step != steps.NameGenerateBody {
			continue
		}
		if text := userPrompt(requests[i]); strings.Contains(text, "Path: "+path+"\n") {
			prompt = text
		}
	}
	if prompt == "" {
		t.Fatalf("the writer was never asked about %s in %d calls", path, script.CallsTo(steps.NameGenerateBody))
	}
	return prompt
}

func sectionOf(prompt, heading string) ([]string, bool) {
	lines := strings.Split(prompt, "\n")
	for i, line := range lines {
		if line != heading && !strings.HasPrefix(line, heading+" ") {
			continue
		}
		out := make([]string, 0, 8)
		for _, held := range lines[i+1:] {
			if held == "" {
				break
			}
			out = append(out, held)
		}
		return out, true
	}
	return nil, false
}

type pictureWant struct {
	path        string
	parentPath  string
	parentTitle string
	instruction string
	titleLine   string
	target      string
	targetUnder string
	products    []string
	planned     bool
}

func assertThePicture(t *testing.T, prompt string, want pictureWant) {
	t.Helper()

	problems := make([]string, 0)
	note := func(held bool, problem string) {
		if !held {
			problems = append(problems, problem)
		}
	}
	note(strings.HasPrefix(prompt, headingPicture+"\n"), "it does not open with "+headingPicture)
	note(!strings.Contains(prompt, headingKeywords), "it carries the pack's keyword section, which the body prompt leaves to its own")

	page, _ := sectionOf(prompt, headingPage)
	note(slices.Contains(page, "Path: "+want.path), headingPage+" does not name "+want.path)
	note(!want.planned || slices.Contains(page, plannedState), headingPage+" does not say the page is planned")

	trail, _ := sectionOf(prompt, headingTrail)
	note(slices.ContainsFunc(trail, func(line string) bool {
		return strings.HasPrefix(line, "- "+want.parentTitle+" ("+want.parentPath)
	}), headingTrail+" does not name the parent "+want.parentTitle+" at "+want.parentPath)

	rules, _ := sectionOf(prompt, headingRules)
	note(slices.Contains(rules, "Field: Body"), headingRules+" does not name the body")
	if want.instruction != "" {
		note(slices.Contains(rules, "Instruction: "+want.instruction), headingRules+" does not carry the scope's instruction")
		note(slices.ContainsFunc(rules, func(line string) bool {
			return strings.HasPrefix(line, "Links: ") && line != noLinks
		}), headingRules+" does not carry the scope's link policy")
	}
	if want.titleLine != "" {
		fields, _ := sectionOf(prompt, headingFields)
		note(slices.Contains(fields, want.titleLine), headingFields+" does not carry "+want.titleLine)
	}
	if len(want.products) > 0 {
		problems = append(problems, productProblems(prompt, want.products)...)
	}
	if want.target != "" {
		problems = append(problems, targetProblems(prompt, want.target, want.targetUnder)...)
	}
	if len(problems) > 0 {
		t.Errorf("the writer's prompt for %s: %s\n%s", want.path, strings.Join(problems, "; "), prompt)
	}
}

func productProblems(prompt string, titles []string) []string {
	listed, held := sectionOf(prompt, headingListed)
	if !held {
		listed, held = sectionOf(prompt, headingTwin)
	}
	if !held {
		return []string{"it names no products, neither " + headingListed + " nor " + headingTwin}
	}
	for _, line := range listed {
		for _, title := range titles {
			if strings.HasPrefix(line, "- "+title+" (") {
				return nil
			}
		}
	}
	return []string{"its products " + strings.Join(listed, " | ") + " name none of the stand's products " + strings.Join(titles, ", ")}
}

func targetProblems(prompt, path, parentTitle string) []string {
	targets, held := sectionOf(prompt, headingTargets)
	if !held {
		return []string{"it names no page to link to"}
	}
	for _, line := range targets {
		if !strings.Contains(line, path+" (") {
			continue
		}
		_, about, found := strings.Cut(line, "About it: ")
		if found && strings.Contains(about, " for ") && strings.Contains(about, " under "+parentTitle) {
			return nil
		}
		return []string{"the target " + path + " reads " + strconv.Quote(line) + ", want a summary built from its structure under " + parentTitle}
	}
	return []string{"no target is " + path + ": " + strings.Join(targets, " | ")}
}

func keepOnSite(t *testing.T, client *wp.Client, path string, wpID int64) string {
	t.Helper()

	ctx := context.WithoutCancel(t.Context())
	original, err := client.GetRaw(ctx, wp.TypePage, wpID)
	if err != nil {
		t.Fatalf("read the raw body of %s: %v", path, err)
	}
	t.Cleanup(func() {
		now, readErr := client.GetRaw(ctx, wp.TypePage, wpID)
		if readErr != nil {
			t.Errorf("read the raw body of %s to put it back: %v", path, readErr)
			return
		}
		if now.Content == original.Content {
			return
		}
		if _, putErr := client.PutRaw(ctx, wp.TypePage, wpID, original.Content, now.ContentHash); putErr != nil {
			t.Errorf("put the client's body of %s back: %v", path, putErr)
		}
	})
	return original.Content
}

func assertTheClientsTextIsKept(t *testing.T, core *app.Core, script *fake.Scripted, client *wp.Client, siteID string,
	page pages.Page, paragraph string) {
	t.Helper()

	if page.WPID == nil {
		t.Fatalf("%s is not on the site", page.Path)
	}
	if original := keepOnSite(t, client, page.Path, *page.WPID); original != paragraph {
		t.Fatalf("%s holds %q, want the stand's paragraph %q; reset the stand", page.Path, original, paragraph)
	}

	runID := startRun(t, core, siteID, run.PublishLive, writing(steps.NamePublish), page.ID)
	awaitRun(t, core.Runs, runID)

	rules, _ := sectionOf(bodyPromptOf(t, script, page.Path), headingRules)
	if !slices.Contains(rules, keptRule) {
		t.Errorf("the rules for %s are %v, want them to keep the client's text", page.Path, rules)
	}
	stored, err := client.GetRaw(t.Context(), wp.TypePage, *page.WPID)
	if err != nil {
		t.Fatalf("read %s back: %v", page.Path, err)
	}
	ours, kept := strings.CutPrefix(stored.Content, paragraph)
	if !kept {
		t.Fatalf("%s now starts %q, want the client's paragraph %q byte for byte", page.Path, stored.Content, paragraph)
	}
	if !strings.Contains(ours, ourText) || strings.Contains(strings.ToLower(ours), "<h1") {
		t.Errorf("after the client's paragraph %s holds %q, want our sections without an H1", page.Path, ours)
	}
}

type clientEdit struct {
	mu     sync.Mutex
	once   sync.Once
	ctx    context.Context
	env    environment
	root   string
	edited string
	err    error
	wpID   int64
	fired  bool
}

func (c *clientEdit) arm(ctx context.Context, wpID int64, edited string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ctx, c.wpID, c.edited = ctx, wpID, edited
}

func (c *clientEdit) reply(req port.Request) string {
	c.once.Do(c.apply)
	return bodyFromPrompt(req)
}

func (c *clientEdit) apply() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fired = true
	if c.wpID == 0 {
		return
	}
	_, c.err = wpRun(c.ctx, c.root, c.env, "post", "update", strconv.FormatInt(c.wpID, 10),
		"--post_content="+c.edited, "--user="+c.env.user)
}

func (c *clientEdit) outcome() (applied bool, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fired && c.wpID != 0, c.err
}

func assertAClientEditPausesThePublish(t *testing.T, core *app.Core, client *wp.Client, edit *clientEdit, siteID string,
	page pages.Page, plan clientEditPlan) {
	t.Helper()

	if page.WPID == nil {
		t.Fatalf("%s is not on the site", page.Path)
	}
	original := keepOnSite(t, client, page.Path, *page.WPID)
	edited := strings.Replace(original, plan.from, plan.to, 1)
	if edited == original {
		t.Fatalf("%s no longer holds %q to edit: %q; reset the stand", page.Path, plan.from, original)
	}
	edit.arm(context.WithoutCancel(t.Context()), *page.WPID, edited)

	runID := startRun(t, core, siteID, run.PublishLive, writing(steps.NamePublish), page.ID)
	awaitPause(t, core.Runs, runID)
	if applied, err := edit.outcome(); !applied || err != nil {
		t.Fatalf("the client's edit of %s while the writer wrote did not land (applied %v): %v", page.Path, applied, err)
	}

	item := itemsByTarget(t, core.Runs, runID)[page.ID]
	if item.Status != string(run.StatusPaused) || item.PauseReason != string(run.PauseNeedsHuman) ||
		item.CurrentStep != steps.NamePublish || !strings.Contains(item.Note, steps.ReasonClientChanged) {
		t.Fatalf("the item for %s is %s/%s at %s with %q, want it paused at publish because the client changed the page%s",
			page.Path, item.Status, item.PauseReason, item.CurrentStep, item.Note, itemStates(t, core.Runs, runID))
	}
	now, err := client.GetRaw(t.Context(), wp.TypePage, *page.WPID)
	if err != nil {
		t.Fatalf("read %s back: %v", page.Path, err)
	}
	if !strings.Contains(now.Content, plan.to) || strings.Contains(now.Content, ourText) {
		t.Errorf("%s holds %q after the pause, want the client's edit and nothing of ours", page.Path, now.Content)
	}
}
