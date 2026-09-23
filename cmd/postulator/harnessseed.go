//go:build uiharness

package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/models"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/schedules"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/application/sync"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	anchorMarker = "PHRASE TO INCLUDE\n"

	siteName  = "Crema Bench"
	pollEvery = 100 * time.Millisecond
	pollFor   = 3 * time.Minute

	failingPath = "/grinders/single-dosing/"

	absentPageID  = "00000000-0000-4000-8000-000000000000"
	relinkedPath  = "/brewing/pre-infusion/"
	revertedPath  = "/accessories/knock-boxes/"
	deniedEntity  = "espresso-blends"
	deniedMessage = "Delete the espresso blends hub and everything under it; nobody searches for it."
	deniedTool    = "graph_delete_subtree"
)

var (
	completedRun = []string{"/brewing/pre-infusion/", "/brewing/channeling/", "/milk-drinks/cortado/", "/beans/freshness/"}
	failedRun    = []string{failingPath}
	runningRun   = []string{"/grinders/flat-burr/", "/grinders/conical-burr/"}
	pausedRun    = []string{"/beans/blends/"}
	cancelledRun = []string{"/beans/single-origin/"}
	revertedRun  = []string{revertedPath}
)

func harnessRecipe() []template.StepSpec {
	return []template.StepSpec{
		{Name: steps.NameResolveContext, Enabled: true},
		{Name: steps.NameGenerateBody, Enabled: true},
		{Name: steps.NameGenerateMeta, Enabled: true},
		{Name: steps.NameInsertLinks, Enabled: true},
		{Name: steps.NameRepairLinks, Enabled: true},
		{Name: steps.NameValidate, Enabled: true},
		{Name: steps.NameJudge, Enabled: true},
		{Name: steps.NamePublish, Enabled: true},
		{Name: steps.NameRelinkNeighbors, Enabled: true},
		{Name: steps.NameSyncBack, Enabled: true},
		{Name: steps.NameReport, Enabled: true},
	}
}

func harnessReplies() []fake.Reply {
	targets := make([]string, 0, 10)
	targets = append(targets, completedRun...)
	targets = append(targets, failedRun...)
	targets = append(targets, runningRun...)
	targets = append(targets, pausedRun...)
	targets = append(targets, cancelledRun...)
	targets = append(targets, revertedRun...)

	out := make([]fake.Reply, 0, 2*len(targets)+2)
	for _, path := range targets {
		subject := subjectOf(path)
		out = append(out,
			fake.Reply{Step: steps.NameGenerateBody, Match: path, Text: draftOf(subject)},
			fake.Reply{Step: steps.NameGenerateMeta, Match: path, Text: metaOf(subject)},
		)
	}
	out = append(out, repairReplies()...)
	out = append(out, proposeReplies()...)
	return append(out,
		fake.Reply{Step: steps.NameJudge, Text: `{"score":0.88,"issues":[],"suggestions":["Add a photograph of the puck after extraction."]}`},
		fake.Reply{Step: "title", Text: "Which topics still need a canonical page"},
	)
}

func proposeReplies() []fake.Reply {
	fromPages := `{"entities":[{"path":"/blog/","name":"The bench journal","kind":"hub","intent":"read what we learned on the bench this month","primaryKeyword":"espresso blog","secondaryKeywords":["coffee journal"],"anchors":["the bench journal"],"parentPath":"","relatedPaths":[]},{"path":"/blog/espresso-for-beginners/","name":"Espresso for beginners","kind":"topic","intent":"survive the first two weeks with a new machine","primaryKeyword":"espresso for beginners","secondaryKeywords":["first espresso machine"],"anchors":["espresso for beginners"],"parentPath":"/blog/","relatedPaths":["/blog/why-your-shot-tastes-sour/"]},{"path":"/blog/why-your-shot-tastes-sour/","name":"Sour shots","kind":"topic","intent":"work out why a shot tastes sour and fix it","primaryKeyword":"sour espresso","secondaryKeywords":["under-extracted shot"],"anchors":["sour shots"],"parentPath":"/blog/","relatedPaths":[]},{"path":"/blog/water-hardness-and-espresso/","name":"Water for espresso","kind":"topic","intent":"choose water that will not ruin the machine","primaryKeyword":"espresso water hardness","secondaryKeywords":["brew water"],"anchors":["water for espresso"],"parentPath":"/blog/","relatedPaths":[]},{"path":"/reviews/","name":"Machine reviews","kind":"hub","intent":"compare the machines we have lived with","primaryKeyword":"espresso machine reviews","secondaryKeywords":["machine review"],"anchors":["machine reviews"],"parentPath":"","relatedPaths":[]},{"path":"/reviews/gaggia-classic-pro/","name":"Gaggia Classic Pro","kind":"product","intent":"decide whether the Classic Pro is the machine to buy","primaryKeyword":"gaggia classic pro","secondaryKeywords":["classic pro review"],"anchors":["the Gaggia Classic Pro"],"parentPath":"/reviews/","relatedPaths":["/reviews/rancilio-silvia/"]},{"path":"/reviews/rancilio-silvia/","name":"Rancilio Silvia","kind":"product","intent":"decide whether the Silvia is the machine to buy","primaryKeyword":"rancilio silvia","secondaryKeywords":["silvia review"],"anchors":["the Rancilio Silvia"],"parentPath":"/reviews/","relatedPaths":["/reviews/gaggia-classic-pro/"]},{"path":"/reviews/lelit-anna/","name":"Lelit Anna","kind":"product","intent":"decide whether the Anna is the machine to buy","primaryKeyword":"lelit anna","secondaryKeywords":["anna review"],"anchors":["the Lelit Anna"],"parentPath":"/reviews/","relatedPaths":[]},{"path":"/glossary/","name":"Espresso glossary","kind":"topic","intent":"look up a word heard in a coffee shop","primaryKeyword":"espresso glossary","secondaryKeywords":["coffee terms"],"anchors":["the glossary"],"parentPath":"","relatedPaths":[]},{"path":"/faq/","name":"Common questions","kind":"topic","intent":"find the answer we give every week","primaryKeyword":"espresso questions","secondaryKeywords":["coffee faq"],"anchors":["the questions we are asked"],"parentPath":"","relatedPaths":[]}]}`

	related := `{"edges":[{"from":"Tampers","to":"Tamping","weight":0.86,"reason":"a tamper is the tool the tamping step is about"},{"from":"Distribution tools","to":"WDT distribution","weight":0.81,"reason":"the tool exists to do the distribution"},{"from":"Espresso scales","to":"Dialing in espresso","weight":0.77,"reason":"dialing in is measured, and the scale is what measures it"},{"from":"Knock boxes","to":"Puck preparation","weight":0.58,"reason":"the puck routine begins by knocking the last one out"},{"from":"Coffee freshness","to":"Espresso blends","weight":0.69,"reason":"a blend is chosen and then raced against its rest date"},{"from":"Espresso roast profiles","to":"Single origin espresso","weight":0.64,"reason":"a single origin is usually roasted on its own profile"}]}`

	return []fake.Reply{
		{Step: graph.NameProposeFromPages, Text: fromPages},
		{Step: graph.NameProposeRelated, Text: related},
	}
}

func repairReplies() []fake.Reply {
	declared := seedEntities()
	out := make([]fake.Reply, 0, len(declared))

	for index := range declared {
		anchor := declared[index].Anchor
		out = append(out, fake.Reply{
			Step:  steps.NameRepairLinks,
			Match: anchorMarker + anchor + "\n",
			Text:  `{"sentence":"We cover ` + anchor + ` in its own guide, and it is worth reading before you change anything here."}`,
		})
	}
	return out
}

func subjectOf(path string) string {
	declared := seedEntities()
	for index := range declared {
		if declared[index].Path == path {
			return declared[index].Keyword
		}
	}
	return strings.Trim(path, "/")
}

func draftOf(subject string) string {
	return `{"title":"How to ` + subject + `: Step-by-Step Guide | ` + siteName + `",` +
		`"h1":"How to ` + subject + `",` +
		`"sections":[` +
		`{"heading":"Introduction","html":"<p>This guide to ` + subject + ` takes about ten minutes to read ` +
		`and assumes you already own a machine, a grinder and a scale that reads to a tenth of a gram.</p>"},` +
		`{"heading":"Step-by-Step Instructions","html":"<p>Weigh eighteen grams into the basket, stir the ` +
		`grounds until no clumps remain, tamp level, lock the portafilter in and start the pump with the ` +
		`scale already zeroed under the cup.</p>"},` +
		`{"heading":"Tips and Common Mistakes","html":"<p>Do not chase a number at the cost of the taste, ` +
		`do not change two variables between shots, and do not judge a bag of beans until it has rested a ` +
		`week past its roast date.</p>"},` +
		`{"heading":"Frequently Asked Questions","html":"<p>Yes, the same routine works on a single boiler ` +
		`machine, and no, a pressurized basket will not tell you whether the puck is prepared well.</p>"}],` +
		`"summary":"A short, practical guide to ` + subject + ` on a home espresso bar."}`
}

func metaOf(subject string) string {
	return `{"title":"How to ` + subject + `: Step-by-Step Guide | ` + siteName + `",` +
		`"description":"What ` + subject + ` changes in the cup, and the routine we use on the bench.",` +
		`"canonical":"","ogTitle":"","ogDescription":""}`
}

type assistantScript struct {
	page atomic.Pointer[string]
	site atomic.Pointer[string]
}

func (a *assistantScript) naming(id string) {
	a.page.Store(&id)
}

func (a *assistantScript) onSite(id string) {
	a.site.Store(&id)
}

func (a *assistantScript) pageID() string {
	if held := a.page.Load(); held != nil {
		return *held
	}
	return ""
}

func (a *assistantScript) siteID() string {
	if held := a.site.Load(); held != nil {
		return *held
	}
	return ""
}

func (a *assistantScript) answer(prompt string) fake.Turn {
	lowered := strings.ToLower(prompt)

	switch {
	case strings.Contains(lowered, "delete"):
		return fake.Turn{
			Tool: deniedTool,
			Args: json.RawMessage(`{"entity":"` + deniedEntity + `"}`),
			Text: "There is no tool that empties a hub in one call, and I will not take the entities out " +
				"one at a time without you saying so. Nothing was read and nothing was written.",
		}
	case strings.Contains(lowered, "open the page"):
		return fake.Turn{
			Tool: "pages_get",
			Args: json.RawMessage(`{"id":"` + absentPageID + `"}`),
			Text: "There is no page with that id on this site. Give me the path instead and I will " +
				"look it up in the map.",
		}
	case strings.Contains(lowered, "every page"):
		return fake.Turn{
			Tool: "pages_list",
			Args: json.RawMessage(`{"siteId":"` + a.siteID() + `","limit":100}`),
			Text: "The listing came back longer than one message can carry, so I read the opening of it. " +
				"Ask me for one hub at a time and I will read each in full.",
		}
	case strings.Contains(lowered, "retitle") || strings.Contains(lowered, "rename"):
		return fake.Turn{
			Tool: "pages_update",
			Args: json.RawMessage(`{"id":"` + a.pageID() + `","metaTitle":"Espresso machines under $500: nine we would buy"}`),
			Text: "I have queued the retitle. The page keeps its path and its canonical, only the meta title " +
				"changes, so the entry in the search results leads with the price the reader searched for.",
		}
	case strings.Contains(lowered, "canonical"):
		return fake.Turn{Text: "Ten entities carry no canonical page yet: the five you planned this week " +
			"under Grinders and Coffee beans, plus Cortado, Channeling, Pre-infusion, Knock boxes and " +
			"Espresso blends. Every one of them already has a planned page mapped to it, so the graph is " +
			"complete and only the drafts are missing."}
	default:
		return fake.Turn{Text: "Your map holds forty-one entities over six hubs, and forty-four of the " +
			"sixty-two pages are mapped. The eighteen that are not are the journal, the reviews and the " +
			"legal pages, which no entity should own."}
	}
}

func seed(ctx context.Context, core *app.Core, site *wptest.Server, provider *pacedProvider,
	script *assistantScript, key string) error {
	created, err := core.Sites.Create(ctx, sites.CreateRequest{
		Name: siteName, BaseURL: site.URL(), Username: harnessUser, Password: harnessPassword, AllowInsecure: true,
	})
	if err != nil {
		return err
	}
	siteID := created.Site.ID

	entities, err := seedGraph(ctx, core, siteID)
	if err != nil {
		return err
	}

	pagesByPath, err := seedPages(ctx, core, siteID, entities)
	if err != nil {
		return err
	}
	script.naming(pagesByPath["/espresso-machines/under-500/"])
	script.onSite(siteID)

	guide, policy, err := seedTemplates(ctx, core, siteID)
	if err != nil {
		return err
	}
	if _, err = core.Sites.Update(ctx, sites.UpdateRequest{
		ID: siteID, Defaults: &sites.Defaults{TemplateID: &guide, LinkPolicyID: &policy},
	}); err != nil {
		return err
	}
	if _, err = core.Graph.RecomputeScores(ctx, graph.RecomputeScoresRequest{SiteID: siteID}); err != nil {
		return err
	}
	if adoptErr := adoptTheSite(ctx, core, site, siteID); adoptErr != nil {
		return adoptErr
	}
	if key != "" {
		_, keyErr := core.Models.SetProviderKey(ctx, models.SetProviderKeyRequest{Provider: "openai", APIKey: key})
		return keyErr
	}
	if runErr := seedRuns(ctx, core, siteID, guide, pagesByPath, provider); runErr != nil {
		return runErr
	}
	if scheduleErr := seedSchedule(ctx, core, siteID, guide); scheduleErr != nil {
		return scheduleErr
	}
	return seedConversation(ctx, core, siteID)
}

func adoptTheSite(ctx context.Context, core *app.Core, site *wptest.Server, siteID string) error {
	if err := placeOnSite(ctx, core, site, siteID); err != nil {
		return err
	}

	started, err := core.Sync.SyncSite(ctx, sync.SyncSiteRequest{SiteID: siteID})
	if err != nil {
		return err
	}
	return waitForRun(ctx, core, started.RunID, run.StatusCompleted)
}

func seedGraph(ctx context.Context, core *app.Core, siteID string) (map[string]string, error) {
	declared := seedEntities()
	byName := make(map[string]string, len(declared))

	for index := range declared {
		entity := &declared[index]
		created, err := core.Graph.CreateEntity(ctx, graph.CreateEntityRequest{
			SiteID: siteID, Name: entity.Name, Kind: entity.Kind, Intent: entity.Intent,
			PrimaryKeyword: entity.Keyword, SecondaryKeywords: []string{entity.Anchor},
			Anchors: []graph.Anchor{{Text: entity.Anchor, Source: "user", Weight: 1}},
			Source:  "user",
		})
		if err != nil {
			return nil, err
		}
		byName[entity.Name] = created.Entity.ID
	}

	for index := range declared {
		entity := &declared[index]
		if entity.Parent == "" {
			continue
		}
		if _, err := core.Graph.AddEdge(ctx, graph.AddEdgeRequest{
			SiteID: siteID, FromEntityID: byName[entity.Name], ToEntityID: byName[entity.Parent],
			Kind: "parent", Weight: 1, Source: "user", Status: "approved",
			Reason: entity.Name + " is one branch of " + entity.Parent + ".",
		}); err != nil {
			return nil, err
		}
	}

	for _, edge := range seedRelated() {
		if _, err := core.Graph.AddEdge(ctx, graph.AddEdgeRequest{
			SiteID: siteID, FromEntityID: byName[edge.From], ToEntityID: byName[edge.To],
			Kind: "related", Weight: 0.7, Source: "ai", Status: edge.Status, Reason: edge.Reason,
		}); err != nil {
			return nil, err
		}
	}
	return byName, nil
}

func seedPages(ctx context.Context, core *app.Core, siteID string, entities map[string]string) (map[string]string, error) {
	byPath := make(map[string]string, 64)
	declared := seedEntities()

	for index := range declared {
		entity := &declared[index]
		owner := entities[entity.Name]
		created, err := core.Pages.Create(ctx, pages.CreateRequest{
			SiteID: siteID, Path: entity.Path, WPType: "page", Title: entity.Title, H1: entity.Title,
			MetaTitle:       entity.Title + " | " + siteName,
			MetaDescription: "What we know about " + entity.Keyword + " after a few hundred shots on the bench.",
			Status:          entity.Status, EntityID: &owner,
		})
		if err != nil {
			return nil, err
		}
		byPath[entity.Path] = created.Page.ID
	}

	for _, page := range seedSecondaryPages() {
		owner := entities[page.Entity]
		created, err := core.Pages.Create(ctx, pages.CreateRequest{
			SiteID: siteID, Path: page.Path, WPType: "post", Title: page.Title, H1: page.Title,
			MetaTitle: page.Title + " | " + siteName, MetaDescription: page.Title + ", written on the bench.",
			Status: page.Status, EntityID: &owner,
		})
		if err != nil {
			return nil, err
		}
		byPath[page.Path] = created.Page.ID
	}

	for _, page := range seedLoosePages() {
		created, err := core.Pages.Create(ctx, pages.CreateRequest{
			SiteID: siteID, Path: page.Path, WPType: "page", Title: page.Title, H1: page.Title,
			MetaTitle: page.Title + " | " + siteName, MetaDescription: page.Title + ".",
			Status: page.Status,
		})
		if err != nil {
			return nil, err
		}
		byPath[page.Path] = created.Page.ID
	}

	for index := range declared {
		if _, err := core.Pages.SetCanonical(ctx, pages.SetCanonicalRequest{
			EntityID: entities[declared[index].Name], PageID: byPath[declared[index].Path],
		}); err != nil {
			return nil, err
		}
	}
	return byPath, nil
}

func seedTemplates(ctx context.Context, core *app.Core, siteID string) (guide, policy string, err error) {
	listed, err := core.Templates.ListTemplates(ctx, templates.ListTemplatesRequest{
		Scope: "global", ListRequest: dto.ListRequest{Limit: 20},
	})
	if err != nil {
		return "", "", err
	}

	for index := range listed.Items {
		if listed.Items[index].PageKind == "guide" {
			guide = listed.Items[index].ID
		}
	}
	if guide == "" {
		return "", "", errors.New(errors.NotFound, "the shipped guide template is missing")
	}

	if _, err = core.Templates.SetOverride(ctx, templates.SetOverrideRequest{
		TemplateID: guide, Scope: "site", TargetID: siteID,
		Patch: json.RawMessage(`{"tone":"Plain and practical; write as a barista talking across the bench, never as a brochure","length":{"min":1200,"max":1800}}`),
	}); err != nil {
		return "", "", err
	}

	created, err := core.Templates.CreatePolicy(ctx, templates.CreatePolicyRequest{
		Scope: "site", SiteID: &siteID, Name: "Crema Bench linking",
		Rules: template.LinkRules{
			UpDepth: 2, DownLinks: true, SiblingMinWeight: 0.6, MaxLinks: 10, MaxPerTarget: 1,
			ParentLinkWithinParagraphs: 2, ChildrenSection: true,
		},
		ForbidExternal: true, ForbidSelf: true, AnchorStrategy: "rotate",
	})
	if err != nil {
		return "", "", err
	}
	return guide, created.Policy.ID, nil
}

func seedSchedule(ctx context.Context, core *app.Core, siteID, guide string) error {
	_, err := core.Schedules.Create(ctx, schedules.CreateRequest{
		SiteID: siteID, Name: "Draft the planned guides every Monday", Cron: "0 6 * * 1",
		Status: "planned", Limit: 5, TemplateID: guide, PublishMode: "draft",
		MaxUSD: 4, Enabled: true,
	})
	return err
}

func seedConversation(ctx context.Context, core *app.Core, siteID string) error {
	opened, err := core.Agent.CreateConversation(ctx, agent.CreateConversationRequest{SiteID: siteID, Mode: "confirm"})
	if err != nil {
		return err
	}
	conversation := opened.Conversation.ID

	if askErr := exchange(ctx, core, conversation, "Which entities still have no canonical page?"); askErr != nil {
		return askErr
	}
	if askErr := exchange(ctx, core, conversation,
		"Open the page 00000000-0000-4000-8000-000000000000 and tell me what it links to."); askErr != nil {
		return askErr
	}
	if askErr := exchange(ctx, core, conversation, "List every page on the site with its status."); askErr != nil {
		return askErr
	}
	if askErr := exchange(ctx, core, conversation, deniedMessage); askErr != nil {
		return askErr
	}
	return exchange(ctx, core, conversation,
		"Retitle the espresso machines under $500 page so the meta title leads with the price.")
}

func exchange(ctx context.Context, core *app.Core, conversation, text string) error {
	before, err := core.Agent.ListMessages(ctx, agent.ListMessagesRequest{
		ConversationID: conversation, ListRequest: dto.ListRequest{Limit: 100},
	})
	if err != nil {
		return err
	}

	if sendErr := waitFor(ctx, "the previous turn to end", func() (bool, error) {
		_, sendErr := core.Agent.Send(ctx, agent.SendRequest{ConversationID: conversation, Text: text})
		if errors.IsCode(sendErr, errors.Conflict) {
			return false, nil
		}
		return sendErr == nil, sendErr
	}); sendErr != nil {
		return sendErr
	}

	return waitFor(ctx, "the agent to answer", func() (bool, error) {
		listed, listErr := core.Agent.ListMessages(ctx, agent.ListMessagesRequest{
			ConversationID: conversation, ListRequest: dto.ListRequest{Limit: 100},
		})
		if listErr != nil {
			return false, listErr
		}
		return len(listed.Items) >= len(before.Items)+2, nil
	})
}

func seedRuns(ctx context.Context, core *app.Core, siteID, guide string, byPath map[string]string, provider *pacedProvider) error {
	completed, err := startRun(ctx, core, siteID, guide, byPath, completedRun)
	if err != nil {
		return err
	}
	if waitErr := waitForRun(ctx, core, completed, run.StatusCompleted); waitErr != nil {
		return waitErr
	}

	provider.failOn(failingPath)
	failed, err := startRun(ctx, core, siteID, guide, byPath, failedRun)
	if err != nil {
		return err
	}
	if waitErr := waitForRun(ctx, core, failed, run.StatusFailed); waitErr != nil {
		return waitErr
	}
	provider.failOn("")

	if relinkErr := seedRelinkRun(ctx, core, siteID, byPath); relinkErr != nil {
		return relinkErr
	}
	if revertErr := seedRevertRun(ctx, core, siteID, guide, byPath); revertErr != nil {
		return revertErr
	}

	provider.hold()
	running, err := startRun(ctx, core, siteID, guide, byPath, runningRun)
	if err != nil {
		return err
	}
	if waitErr := waitForRun(ctx, core, running, run.StatusRunning); waitErr != nil {
		return waitErr
	}

	paused, err := startRun(ctx, core, siteID, guide, byPath, pausedRun)
	if err != nil {
		return err
	}
	if _, err = core.Runs.Pause(ctx, runs.PauseRequest{RunID: paused, Reason: string(run.PauseUser)}); err != nil {
		return err
	}

	cancelled, err := startRun(ctx, core, siteID, guide, byPath, cancelledRun)
	if err != nil {
		return err
	}
	_, err = core.Runs.Cancel(ctx, runs.CancelRequest{RunID: cancelled})
	return err
}

func seedRelinkRun(ctx context.Context, core *app.Core, siteID string, byPath map[string]string) error {
	target, known := byPath[relinkedPath]
	if !known {
		return errors.New(errors.NotFound, "the harness seeded no page at "+relinkedPath)
	}

	started, err := core.Runs.Start(ctx, runs.StartRequest{
		SiteID: siteID, PageIDs: []string{target}, Kind: string(run.KindRelink),
	})
	if err != nil {
		return err
	}
	return waitForRun(ctx, core, started.RunID, run.StatusCompleted)
}

func seedRevertRun(ctx context.Context, core *app.Core, siteID, guide string, byPath map[string]string) error {
	source, err := startRun(ctx, core, siteID, guide, byPath, revertedRun)
	if err != nil {
		return err
	}
	if waitErr := waitForRun(ctx, core, source, run.StatusCompleted); waitErr != nil {
		return waitErr
	}

	started, err := core.Runs.Revert(ctx, runs.RevertRequest{RunID: source})
	if err != nil {
		return err
	}
	return waitForRun(ctx, core, started.RunID, run.StatusCompleted)
}

func startRun(ctx context.Context, core *app.Core, siteID, guide string, byPath map[string]string, paths []string) (string, error) {
	targets := make([]string, 0, len(paths))
	for _, path := range paths {
		id, known := byPath[path]
		if !known {
			return "", errors.New(errors.NotFound, "the harness seeded no page at "+path)
		}
		targets = append(targets, id)
	}

	started, err := core.Runs.Start(ctx, runs.StartRequest{
		SiteID: siteID, PageIDs: targets, TemplateID: guide,
		PublishMode: string(run.PublishDraft), Recipe: harnessRecipe(),
	})
	if err != nil {
		return "", err
	}
	return started.RunID, nil
}

func waitForRun(ctx context.Context, core *app.Core, runID string, want run.Status) error {
	return waitFor(ctx, "run "+runID+" to be "+string(want), func() (bool, error) {
		got, err := core.Runs.Get(ctx, runs.GetRequest{RunID: runID})
		if err != nil {
			return false, err
		}
		if got.Run.Status == string(want) {
			return true, nil
		}

		settled := run.Status(got.Run.Status)
		if !settled.Terminal() && settled != run.StatusPaused {
			return false, nil
		}
		return false, errors.New(errors.Conflict, "the run settled as "+got.Run.Status+
			" rather than "+string(want)+": "+got.Run.PauseReason+" "+got.Run.Error+
			itemStates(ctx, core, runID)).WithDetail("runId", runID)
	})
}

func itemStates(ctx context.Context, core *app.Core, runID string) string {
	listed, err := core.Runs.ListItems(ctx, runs.ListItemsRequest{
		RunID: runID, ListRequest: dto.ListRequest{Limit: 50},
	})
	if err != nil {
		return "the items could not be listed: " + err.Error()
	}

	out := strings.Builder{}
	for i := range listed.Items {
		item := listed.Items[i]
		out.WriteString("; " + item.Status + " at " + item.CurrentStep + " " +
			item.PauseReason + " " + item.Error)
	}
	return out.String()
}

func waitFor(ctx context.Context, what string, done func() (bool, error)) error {
	started := time.Now()
	defer func() { log.Printf("harness waited %s for %s", time.Since(started).Round(time.Millisecond), what) }()

	deadline := time.Now().Add(pollFor)
	for time.Now().Before(deadline) {
		ready, err := done()
		if err != nil {
			return err
		}
		if ready {
			return nil
		}

		select {
		case <-ctx.Done():
			return errors.New(errors.Cancelled, "the harness stopped waiting for "+what).WithInternal(ctx.Err())
		case <-time.After(pollEvery):
		}
	}
	return errors.New(errors.Internal, "the harness timed out waiting for "+what)
}
