package tools

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	runsControlName = "runs_control"
	opStart         = "start"
	opEstimate      = "estimate"
	opPause         = "pause"
	opResume        = "resume"
	opCancel        = "cancel"
	opRetryStep     = "retry_step"
	opRegenerate    = "regenerate"
	opRevert        = "revert"
	maxPagesNamed   = 3
)

type briefArgs struct {
	Goal        string `json:"goal,omitempty" description:"What the run is for, in a line"`
	Focus       string `json:"focus,omitempty" description:"What every page should dwell on"`
	Tone        string `json:"tone,omitempty" description:"How it should sound"`
	Avoid       string `json:"avoid,omitempty" description:"What to stay away from"`
	Reference   string `json:"reference,omitempty" description:"Text the writer draws facts from, up to 20,000 characters"`
	StyleSample string `json:"styleSample,omitempty" description:"A passage whose style to follow, up to 4,000 characters"`
}

type mustLinkArgs struct {
	PageID string `json:"pageId" description:"The page to link to; the entity graph must connect them, or addEdges"`
	Anchor string `json:"anchor,omitempty" description:"The anchor text"`
}

type directiveArgs struct {
	PageID      string         `json:"pageId" description:"One of the run's pages"`
	Notes       string         `json:"notes,omitempty" description:"What this page needs"`
	AddSections []string       `json:"addSections,omitempty" description:"Section headings to add, at most 10"`
	Keywords    []string       `json:"keywords,omitempty" description:"Phrases to use, at most 20"`
	Anchors     []string       `json:"anchors,omitempty" description:"Anchors to prefer for its links"`
	MustLink    []mustLinkArgs `json:"mustLink,omitempty" description:"Links the page must carry"`
	MinWords    int            `json:"minWords,omitempty" minimum:"0" description:"The body's least words for this run"`
	MaxWords    int            `json:"maxWords,omitempty" minimum:"0" description:"The body's most words for this run"`
}

type runsControlArgs struct {
	Op             string          `json:"op" enum:"start,estimate,pause,resume,cancel,retry_step,regenerate,revert" description:"start answers a run id at once; estimate prices a start; revert undoes a finished run"`
	PageIDs        []string        `json:"pageIds,omitempty" description:"start, estimate: the pages, or give select"`
	Select         *selectArgs     `json:"select,omitempty" description:"start, estimate: the pages by subtree or type"`
	TemplateID     string          `json:"templateId,omitempty" description:"start, estimate: the template to move the pages to; default their own"`
	Steps          []string        `json:"steps,omitempty" enum:"resolve_context,generate_body,generate_meta,generate_fields,insert_links,repair_links,generate_images,validate,judge,publish,relink_neighbors,sync_back,report" description:"start, estimate: the steps in order; default the template's recipe"`
	Fields         []string        `json:"fields,omitempty" description:"start, estimate: the field keys the run writes, its other steps skipped; revert: the fields to put back; default every field"`
	Rewrite        []string        `json:"rewrite,omitempty" description:"start, estimate: page ids whose owner's text the run may rewrite"`
	PublishMode    string          `json:"publishMode,omitempty" enum:"draft,publish" description:"start, estimate: default draft"`
	Kind           string          `json:"kind,omitempty" enum:"generate,relink,audit,sync,import,custom" description:"start, estimate: the run's purpose; default from the steps"`
	MaxUSD         float64         `json:"maxUsd,omitempty" minimum:"0" description:"start, estimate: dollar ceiling; default configured"`
	MaxTokens      int             `json:"maxTokens,omitempty" minimum:"0" description:"start, estimate: token ceiling; default configured"`
	Brief          *briefArgs      `json:"brief,omitempty" description:"start, estimate: what the whole run is for; the writer, the linker and the judge read it first"`
	Directives     []directiveArgs `json:"directives,omitempty" description:"start, estimate: what one page of the run needs"`
	AddEdges       bool            `json:"addEdges,omitempty" description:"start: add an approved related edge for each mustLink the graph does not connect"`
	RunID          string          `json:"runId,omitempty" description:"pause, resume, cancel, regenerate, revert: the run"`
	Reason         string          `json:"reason,omitempty" enum:"budget_exceeded,awaiting_confirmation,needs_human,user,awaiting_parent" description:"pause: why; default user"`
	ItemID         string          `json:"itemId,omitempty" description:"retry_step: the stopped item, from runs items"`
	AcceptFindings bool            `json:"acceptFindings,omitempty" description:"retry_step: let a page held at validate go on as it is"`
	ItemIDs        []string        `json:"itemIds,omitempty" description:"regenerate: stopped items; one that already wrote to the site is refused"`
	PageID         string          `json:"pageId,omitempty" description:"revert: one page of the run to put back; default every page"`
}

func runsControl(deps Deps) Tool {
	tool := newSiteTool(Def{
		Name: runsControlName,
		Description: "Start or price a run over pages, with a brief for the whole run and a directive per page, or " +
			"pause, resume, cancel, retry, regenerate or revert one.",
		Risk: RiskDangerous,
	}, func(ctx context.Context, b Binding, in runsControlArgs) (any, error) {
		return deps.controlRun(ctx, b, &in)
	})
	tool = describing(tool, func(ctx context.Context, b Binding, in runsControlArgs) (string, error) {
		return deps.runChange(ctx, b, &in)
	})
	tool.Price = func(ctx context.Context, b Binding, args json.RawMessage) (Price, error) {
		var in runsControlArgs
		if err := json.Unmarshal(args, &in); err != nil || in.Op != opStart {
			return Price{}, nil
		}
		req, err := deps.startRequest(ctx, b, &in)
		if err != nil {
			return Price{}, err
		}
		priced, err := deps.Runs.Estimate(ctx, req)
		if err != nil {
			return Price{}, err
		}
		return Price{USD: priced.Estimate.USD, Tokens: priced.Estimate.Tokens}, nil
	}
	tool.Edits = func(ctx context.Context, _ Binding, args json.RawMessage) ([]agent.EditField, error) {
		var in runsControlArgs
		if err := json.Unmarshal(args, &in); err != nil || in.Op != opStart {
			return nil, nil
		}
		return deps.startEdits(ctx, &in)
	}
	return riskedBy(tool, map[string]Risk{
		opStart: RiskWrite, opEstimate: RiskRead, opPause: RiskWrite, opResume: RiskWrite, opCancel: RiskWrite,
		opRetryStep: RiskWrite, opRegenerate: RiskWrite,
	})
}

func (deps Deps) controlRun(ctx context.Context, b Binding, in *runsControlArgs) (any, error) {
	switch in.Op {
	case opStart, opEstimate:
		req, err := deps.startRequest(ctx, b, in)
		if err != nil {
			return nil, err
		}
		if in.Op == opEstimate {
			return deps.Runs.Estimate(ctx, req)
		}
		return deps.Runs.Start(ctx, req)
	case opPause:
		return deps.Runs.Pause(ctx, runs.PauseRequest{RunID: in.RunID, Reason: in.Reason})
	case opResume:
		return deps.Runs.Resume(ctx, runs.ResumeRequest{RunID: in.RunID})
	case opCancel:
		return deps.Runs.Cancel(ctx, runs.CancelRequest{RunID: in.RunID})
	case opRetryStep:
		return deps.Runs.RetryStep(ctx, runs.RetryStepRequest{ItemID: in.ItemID, AcceptFindings: in.AcceptFindings})
	case opRegenerate:
		return deps.Runs.Regenerate(ctx, runs.RegenerateRequest{RunID: in.RunID, ItemIDs: in.ItemIDs})
	case opRevert:
		return deps.Runs.Revert(ctx, runs.RevertRequest{RunID: in.RunID, PageID: in.PageID, Fields: in.Fields})
	default:
		return nil, errors.New(errors.Invalid, "runs_control takes the op start, estimate, pause, resume, cancel, retry_step, "+
			"regenerate or revert").WithDetail("field", "op")
	}
}

func (deps Deps) startRequest(ctx context.Context, b Binding, in *runsControlArgs) (runs.StartRequest, error) {
	pageIDs := in.PageIDs
	if len(pageIDs) == 0 && !in.Select.empty() {
		picked, err := deps.selectPages(ctx, b, in.Select)
		if err != nil {
			return runs.StartRequest{}, err
		}
		pageIDs = picked.ids
	}
	req := runs.StartRequest{
		SiteID: b.SiteID, PageIDs: pageIDs, TemplateID: in.TemplateID, Recipe: recipeOf(in.Steps), PublishMode: in.PublishMode,
		Kind: in.Kind, Budget: run.Budget{MaxUSD: in.MaxUSD, MaxTokens: in.MaxTokens}, Rewrite: in.Rewrite, Fields: in.Fields,
		AddEdges: in.AddEdges,
	}
	if in.Brief != nil {
		req.Brief = &run.Brief{
			Goal: in.Brief.Goal, Focus: in.Brief.Focus, Tone: in.Brief.Tone, Avoid: in.Brief.Avoid,
			Reference: in.Brief.Reference, StyleSample: in.Brief.StyleSample,
		}
	}
	for i := range in.Directives {
		held := &in.Directives[i]
		directive := run.Directive{
			PageID: held.PageID, Notes: held.Notes, AddSections: held.AddSections, Keywords: held.Keywords, Anchors: held.Anchors,
			MinWords: held.MinWords, MaxWords: held.MaxWords,
		}
		for _, link := range held.MustLink {
			directive.MustLink = append(directive.MustLink, run.MustLink{PageID: link.PageID, Anchor: link.Anchor})
		}
		req.Directives = append(req.Directives, directive)
	}
	return req, nil
}

func (deps Deps) startEdits(ctx context.Context, in *runsControlArgs) ([]agent.EditField, error) {
	out := []agent.EditField{
		{Name: "goal", Path: "brief.goal"}, {Name: "focus", Path: "brief.focus"},
		{Name: "tone", Path: "brief.tone"}, {Name: "avoid", Path: "brief.avoid"},
	}
	for i := range in.Directives {
		held := &in.Directives[i]
		path, err := deps.pathOf(ctx, held.PageID)
		if err != nil {
			return nil, err
		}
		at := "directives." + strconv.Itoa(i)
		out = append(out, agent.EditField{Name: "notes for " + path, Path: at + ".notes"})
		for j, link := range held.MustLink {
			target, targetErr := deps.pathOf(ctx, link.PageID)
			if targetErr != nil {
				return nil, targetErr
			}
			out = append(out, agent.EditField{Name: "anchor from " + path + " to " + target, Path: at + ".mustLink." + strconv.Itoa(j) + ".anchor"})
		}
	}
	return out, nil
}

func (deps Deps) runChange(ctx context.Context, b Binding, in *runsControlArgs) (string, error) {
	switch in.Op {
	case opStart:
		req, err := deps.startRequest(ctx, b, in)
		if err != nil {
			return "", err
		}
		priced, err := deps.Runs.Estimate(ctx, req)
		if err != nil {
			return "", err
		}
		return deps.startSummary(ctx, in, req.PageIDs, priced)
	case opPause:
		return "Pause the run " + in.RunID, nil
	case opResume:
		return "Resume the run " + in.RunID, nil
	case opCancel:
		return "Cancel the run " + in.RunID + " and every item still in it", nil
	case opRetryStep:
		if in.AcceptFindings {
			return "Let the item " + in.ItemID + " go on past validate as it is", nil
		}
		return "Retry the stopped step of the item " + in.ItemID, nil
	case opRegenerate:
		return "Write " + plural(len(in.ItemIDs), "stopped item") + " of the run " + in.RunID + " again from their first step", nil
	default:
		said := "Undo what the run " + in.RunID + " wrote"
		if in.PageID != "" {
			path, err := deps.pathOf(ctx, in.PageID)
			if err != nil {
				return "", err
			}
			said += " to " + path
		}
		if len(in.Fields) > 0 {
			said += ", its " + strings.Join(in.Fields, ", ") + " only"
		}
		return said + ": created pages go to the trash and written fields and blocks get back what they held", nil
	}
}

func (deps Deps) startSummary(ctx context.Context, in *runsControlArgs, pageIDs []string, priced runs.EstimateResponse) (string, error) {
	named := make([]string, 0, min(len(pageIDs), maxPagesNamed))
	for _, pageID := range pageIDs[:min(len(pageIDs), maxPagesNamed)] {
		path, err := deps.pathOf(ctx, pageID)
		if err != nil {
			return "", err
		}
		named = append(named, path)
	}
	pages := strings.Join(named, ", ")
	if more := len(pageIDs) - len(named); more > 0 {
		pages += " and " + strconv.Itoa(more) + " more"
	}
	said := "Start a " + kindOf(in) + " run over " + plural(len(pageIDs), "page") + " (" + pages + ")"
	if len(in.Fields) > 0 {
		said += " writing only " + strings.Join(in.Fields, ", ")
	}
	if len(in.Rewrite) > 0 {
		said += ", rewriting the owner's own text on " + plural(len(in.Rewrite), "page")
	}
	if in.Brief != nil && in.Brief.Goal != "" {
		said += ", to " + quoted(in.Brief.Goal)
	}
	if len(in.Directives) > 0 {
		said += ", with a directive for " + plural(len(in.Directives), "page")
	}
	if len(priced.Added) > 0 {
		said += ", creating " + plural(len(priced.Added), "planned parent") + " first"
	}
	mode := "as drafts that wait for a review"
	if in.PublishMode == string(run.PublishLive) {
		mode = "straight to the site"
	}
	return said + ", " + mode + ": estimated $" + strconv.FormatFloat(priced.Estimate.USD, 'f', 2, 64) + " for " +
		strconv.Itoa(priced.Estimate.Tokens) + " tokens", nil
}

func kindOf(in *runsControlArgs) string {
	if in.Kind != "" {
		return in.Kind
	}
	return string(run.KindGenerate)
}

func plural(count int, noun string) string {
	if count == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(count) + " " + noun + "s"
}

func recipeOf(steps []string) []template.StepSpec {
	if len(steps) == 0 {
		return nil
	}

	recipe := make([]template.StepSpec, 0, len(steps))
	for _, step := range steps {
		recipe = append(recipe, template.StepSpec{Name: step, Enabled: true})
	}
	return recipe
}
