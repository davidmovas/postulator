//go:build uiharness

package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/schedules"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	siteName  = "Crema Bench"
	pollEvery = 100 * time.Millisecond
	pollFor   = 3 * time.Minute

	failingPath = "/grinders/single-dosing/"
)

var (
	completedRun = []string{"/brewing/pre-infusion/", "/brewing/channeling/", "/milk-drinks/cortado/", "/beans/freshness/", "/accessories/knock-boxes/"}
	failedRun    = []string{failingPath}
	runningRun   = []string{"/grinders/flat-burr/", "/grinders/conical-burr/"}
	pausedRun    = []string{"/beans/blends/"}
	cancelledRun = []string{"/beans/single-origin/"}
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

	out := make([]fake.Reply, 0, 2*len(targets)+2)
	for _, path := range targets {
		subject := subjectOf(path)
		out = append(out,
			fake.Reply{Step: steps.NameGenerateBody, Match: path, Text: draftOf(subject)},
			fake.Reply{Step: steps.NameGenerateMeta, Match: path, Text: metaOf(subject)},
		)
	}
	return append(out,
		fake.Reply{Step: steps.NameJudge, Text: `{"score":0.88,"issues":[],"suggestions":["Add a photograph of the puck after extraction."]}`},
		fake.Reply{Step: steps.NameRepairLinks, Text: `{"sentence":"It sits beside the rest of our espresso brewing technique guides."}`},
	)
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
	page string
}

func (a *assistantScript) answer(prompt string) fake.Turn {
	lowered := strings.ToLower(prompt)

	switch {
	case strings.Contains(lowered, "retitle") || strings.Contains(lowered, "rename"):
		return fake.Turn{
			Tool: "pages_update",
			Args: json.RawMessage(`{"id":"` + a.page + `","metaTitle":"Espresso machines under $500: nine we would buy"}`),
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

func seed(ctx context.Context, core *app.Core, baseURL string, provider *pacedProvider, script *assistantScript) error {
	site, err := core.Sites.Create(ctx, sites.CreateRequest{
		Name: siteName, BaseURL: baseURL, Username: harnessUser, Password: harnessPassword, AllowInsecure: true,
	})
	if err != nil {
		return err
	}
	siteID := site.Site.ID

	entities, err := seedGraph(ctx, core, siteID)
	if err != nil {
		return err
	}

	pagesByPath, err := seedPages(ctx, core, siteID, entities)
	if err != nil {
		return err
	}
	script.page = pagesByPath["/espresso-machines/under-500/"]

	guide, policy, err := seedTemplates(ctx, core, siteID)
	if err != nil {
		return err
	}
	if _, err = core.Sites.Update(ctx, sites.UpdateRequest{
		ID: siteID, Defaults: &sites.Defaults{TemplateID: &guide, LinkPolicyID: &policy},
	}); err != nil {
		return err
	}
	if runErr := seedRuns(ctx, core, siteID, guide, pagesByPath, provider); runErr != nil {
		return runErr
	}
	if scheduleErr := seedSchedule(ctx, core, siteID, guide); scheduleErr != nil {
		return scheduleErr
	}
	return seedConversation(ctx, core, siteID)
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
			SiteID: siteID, FromEntityID: byName[entity.Parent], ToEntityID: byName[entity.Name],
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
		return got.Run.Status == string(want), nil
	})
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
