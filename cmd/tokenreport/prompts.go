package main

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/adapters/llm/catalog"
	"github.com/davidmovas/postulator/internal/adapters/llm/tuning"
	appcontent "github.com/davidmovas/postulator/internal/application/content"
	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/application/sitemodel/sitemodeltest"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/settings"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const reportRun = "token-report"

type prompt struct {
	Model        string    `json:"model"`
	Instructions measure   `json:"instructions"`
	Pack         measure   `json:"pack"`
	Sections     []measure `json:"sections"`
	Rest         measure   `json:"rest"`
	Schema       measure   `json:"schema"`
	Total        measure   `json:"total"`
}

type linkerRow struct {
	Call   *prompt `json:"call,omitempty"`
	Calls  int     `json:"calls"`
	Tries  int     `json:"tries"`
	Priced int     `json:"priced"`
}

type pageRow struct {
	Stand   string     `json:"stand"`
	Path    string     `json:"path"`
	Type    string     `json:"type"`
	Skipped string     `json:"skipped,omitempty"`
	NoBody  bool       `json:"noBody"`
	Writer  *prompt    `json:"writer,omitempty"`
	Linker  *linkerRow `json:"linker,omitempty"`
	Judge   *prompt    `json:"judge,omitempty"`
}

type harness struct {
	model    *sitemodeltest.Memory
	specs    *templates.Service
	client   *recordingClient
	pictures *recordedPictures
	deps     steps.Deps
}

func newHarness(model *sitemodeltest.Memory, models *catalog.Catalog) *harness {
	client := &recordingClient{}
	pictures := &recordedPictures{service: model.Service()}
	profiles := catalogProfiles{models: models}
	pages, entities, site, types := capturedPages{model}, capturedEntities{model}, capturedSite{model}, capturedTypes{model}
	return &harness{
		model: model, client: client, pictures: pictures,
		specs: templates.New(seededTemplates(), pages, entities, site, types, readOnly{}, readOnly{}, clock.System{}),
		deps: steps.Deps{
			Entities: entities, Edges: capturedEdges{model}, Pages: pages, Sites: site, Types: types, Fields: capturedFields{model},
			Profiles: profiles, Tuning: tuning.NewPolicy(settings.Default().NewValues()), Models: models, Pictures: pictures,
			Content: appcontent.New(appcontent.Deps{Profiles: profiles, LLM: client}), LLM: client,
		},
	}
}

func measurePages(ctx context.Context, c *counters, models *catalog.Catalog) ([]pageRow, error) {
	rows := make([]pageRow, 0)
	for _, name := range sitemodeltest.Stands() {
		model, err := typedStand(name)
		if err != nil {
			return nil, err
		}
		h := newHarness(model, models)
		pages := slices.DeleteFunc(slices.Clone(model.Pages), func(page pagemap.Page) bool { return page.Status == pagemap.StatusArchived })
		slices.SortFunc(pages, func(a, b pagemap.Page) int { return cmp.Compare(a.Path, b.Path) })
		for i := range pages {
			row, err := h.measure(ctx, c, name, pages[i])
			if err != nil {
				return nil, fmt.Errorf("%s %s: %w", name, pages[i].Path, err)
			}
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func (h *harness) measure(ctx context.Context, c *counters, stand string, page pagemap.Page) (pageRow, error) {
	kind := h.model.Types.TypeOf(page)
	row := pageRow{Stand: stand, Path: page.Path, Type: kind.Name}
	resolved, err := h.specs.ResolveForPage(ctx, templates.ResolveForPageRequest{PageID: page.ID})
	if errors.IsCode(err, errors.NotFound) {
		row.Skipped = "no template: a run writes no " + kind.Name + " page"
		return row, nil
	}
	if err != nil {
		return pageRow{}, err
	}
	if page.EntityID == nil {
		row.Skipped = "unmapped: the steps refuse a page with no entity"
		return row, nil
	}

	sc := &run.StepContext{
		Run:  run.Run{ID: reportRun, SiteID: page.SiteID, Kind: run.KindGenerate},
		Item: run.Item{ID: reportRun + ":" + page.ID, RunID: reportRun, SiteID: page.SiteID, TargetID: page.ID},
		Page: page, Spec: resolved.Spec, Type: kind, Links: h.model.Types.Policy(page),
		Params: map[string]any{}, Artifacts: map[run.ArtifactKind]run.Artifact{}, Check: run.NewCheckpoint(),
	}
	links, err := steps.ResolveContext(h.deps).Run(ctx, sc)
	if err != nil {
		return pageRow{}, err
	}
	hold(sc, links.Artifacts...)

	if row.Writer, err = h.asked(ctx, c, steps.GenerateBody(h.deps), sc, contextpack.SectionKeywords); err != nil {
		return pageRow{}, err
	}

	raw := currentBody(h.model, page.ID)
	row.NoBody = strings.TrimSpace(raw) == ""
	hold(sc, run.Artifact{Kind: run.ArtifactBodyHTML, Blob: []byte(raw)})

	if row.Linker, err = h.linker(ctx, c, sc); err != nil {
		return pageRow{}, err
	}
	if row.Judge, err = h.asked(ctx, c, steps.Judge(h.deps), sc, contextpack.SectionCurrent, contextpack.SectionKeywords); err != nil {
		return pageRow{}, err
	}
	return row, nil
}

func (h *harness) asked(ctx context.Context, c *counters, def run.StepDef, sc *run.StepContext, leave ...contextpack.SectionName) (*prompt, error) {
	h.client.take()
	h.pictures.take()
	_, stepErr := def.Run(ctx, sc)
	asked, seen := h.client.take(), h.pictures.take()
	if len(asked) == 0 && stepErr != nil {
		return nil, fmt.Errorf("%s sent no request: %w", def.Name, stepErr)
	}
	if len(asked) == 0 {
		return nil, fmt.Errorf("%s returned without sending a request", def.Name)
	}
	if len(seen) == 0 {
		return nil, fmt.Errorf("%s read no pack", def.Name)
	}
	measured, err := c.promptOf(asked[0], &seen[len(seen)-1], leave...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", def.Name, err)
	}
	return &measured, nil
}

func (h *harness) linker(ctx context.Context, c *counters, sc *run.StepContext) (*linkerRow, error) {
	def := steps.RepairLinks(h.deps)
	h.client.take()
	h.pictures.take()
	_, stepErr := def.Run(ctx, sc)
	asked := h.client.take()
	h.pictures.take()
	row := &linkerRow{
		Calls: len(asked), Tries: tries(sc.Params),
		Priced: def.Price.Calls(run.Target{Page: sc.Page, Spec: sc.Spec, Links: sc.Links}, sc.Params),
	}
	if stepErr != nil {
		return nil, fmt.Errorf("%s: %w", def.Name, stepErr)
	}
	if len(asked) == 0 {
		return row, nil
	}
	call, err := c.promptOf(asked[0], nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", def.Name, err)
	}
	row.Call = &call
	return row, nil
}

func tries(params map[string]any) int {
	if count, ok := params[steps.ParamIterations].(float64); ok {
		return int(count)
	}
	return defaultTries
}

const defaultTries = 2

func hold(sc *run.StepContext, artifacts ...run.Artifact) {
	for i := range artifacts {
		held := artifacts[i]
		held.Size, held.Hash = len(held.Blob), run.HashBlob(held.Blob)
		sc.Artifacts[held.Kind] = held
	}
}

func currentBody(model *sitemodeltest.Memory, pageID string) string {
	for i := range model.Bodies {
		if model.Bodies[i].PageID == pageID && model.Bodies[i].Side == block.Current {
			return model.Bodies[i].Raw
		}
	}
	return ""
}

func (cs *counters) promptOf(req llm.Request, seen *sitemodel.Picture, leave ...contextpack.SectionName) (prompt, error) {
	c, err := cs.of(req.Ref)
	if err != nil {
		return prompt{}, err
	}
	if len(req.Messages) != 1 {
		return prompt{}, fmt.Errorf("the request carries %d messages, want the one user message a step sends", len(req.Messages))
	}
	user := req.Messages[0].Text
	for _, placeholder := range template.Placeholders() {
		if strings.Contains(req.System, placeholder) || strings.Contains(user, placeholder) {
			return prompt{}, fmt.Errorf("the placeholder %s reached the prompt unexpanded", placeholder)
		}
	}
	schema, err := json.Marshal(req.Schema)
	if err != nil {
		return prompt{}, err
	}

	measured := prompt{Model: req.Ref.Model}
	if measured.Instructions, err = c.measure("instructions", req.System); err != nil {
		return prompt{}, err
	}
	if measured.Schema, err = c.measure("schema", string(schema)); err != nil {
		return prompt{}, err
	}
	whole, err := c.measure("the user message", user)
	if err != nil {
		return prompt{}, err
	}
	if seen != nil {
		if err := c.packOf(&measured, *seen, user, leave); err != nil {
			return prompt{}, err
		}
	}
	measured.Rest = measure{Name: "the rest", Bytes: whole.Bytes - measured.Pack.Bytes, Tokens: whole.Tokens - measured.Pack.Tokens}
	measured.Total = sum("total", measured.Instructions, whole, measured.Schema)
	return measured, nil
}

func (c counter) packOf(measured *prompt, seen sitemodel.Picture, user string, leave []contextpack.SectionName) error {
	kept := seen.Fitted
	kept.Sections = slices.DeleteFunc(slices.Clone(seen.Fitted.Sections), func(section contextpack.Section) bool {
		return slices.Contains(leave, section.Name)
	})
	text := kept.Text()
	if !strings.Contains(user, text) {
		return errors.New(errors.Internal, "the pack the step read is not in its prompt as this report renders it")
	}
	var err error
	if measured.Pack, err = c.measure("pack", text); err != nil {
		return err
	}
	for i := range kept.Sections {
		section, err := c.measure(string(kept.Sections[i].Name), contextpack.Fitted{Sections: kept.Sections[i : i+1]}.Text())
		if err != nil {
			return err
		}
		measured.Sections = append(measured.Sections, section)
	}
	return nil
}

func writePages(out io.Writer, rows []pageRow) error {
	return tables(out, func(s *sheet) {
		s.heading("THE STEPS, PER STAND PAGE (tokens)", "step", "model", "instructions", "pack", "rest", "schema", "total", "calls")
		for i := range rows {
			row := &rows[i]
			if row.Skipped != "" {
				continue
			}
			name := row.Stand + " " + row.Path + " (" + row.Type + ")"
			if row.NoBody {
				name += ", no body"
			}
			s.prompt(name, "writer", row.Writer, "1")
			owed := strconv.Itoa(row.Linker.Calls/max(row.Linker.Tries, 1)) + " owed, " + strconv.Itoa(row.Linker.Priced) + " priced"
			if row.Linker.Call == nil {
				s.row("", "linker", "", "", "", "", "", "", owed)
			} else {
				s.prompt("", "linker", row.Linker.Call, owed)
			}
			s.prompt("", "judge", row.Judge, "1")
		}
	}, func(s *sheet) {
		s.heading("NOT MEASURED", "why")
		for i := range rows {
			if rows[i].Skipped != "" {
				s.row(rows[i].Stand+" "+rows[i].Path+" ("+rows[i].Type+")", rows[i].Skipped)
			}
		}
	}, func(s *sheet) {
		s.heading("THE PACK, SECTION BY SECTION (tokens)", "step", "sections")
		for i := range rows {
			row := &rows[i]
			if row.Skipped != "" {
				continue
			}
			s.row(row.Stand+" "+row.Path, "writer", sectionsOf(row.Writer))
			s.row("", "judge", sectionsOf(row.Judge))
		}
	})
}

func (s *sheet) prompt(name, step string, p *prompt, calls string) {
	s.row(name, step, p.Model, strconv.Itoa(p.Instructions.Tokens), strconv.Itoa(p.Pack.Tokens), strconv.Itoa(p.Rest.Tokens),
		strconv.Itoa(p.Schema.Tokens), strconv.Itoa(p.Total.Tokens), calls)
}

func sectionsOf(p *prompt) string {
	said := make([]string, 0, len(p.Sections))
	for _, section := range p.Sections {
		said = append(said, section.Name+" "+strconv.Itoa(section.Tokens))
	}
	return strings.Join(said, " · ")
}
