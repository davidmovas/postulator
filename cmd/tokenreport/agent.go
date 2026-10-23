package main

import (
	"cmp"
	"context"
	"encoding/json"
	"io"
	"slices"
	"strconv"

	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/sitemodel/sitemodeltest"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/template"
)

type measure struct {
	Name   string `json:"name"`
	Bytes  int    `json:"bytes"`
	Tokens int    `json:"tokens"`
}

func (c counter) measure(name, text string) (measure, error) {
	tokens, err := c.count(text)
	if err != nil {
		return measure{}, err
	}
	return measure{Name: name, Bytes: len(text), Tokens: tokens}, nil
}

func sum(name string, parts ...measure) measure {
	total := measure{Name: name}
	for _, part := range parts {
		total.Bytes += part.Bytes
		total.Tokens += part.Tokens
	}
	return total
}

type agentReport struct {
	Instructions []measure    `json:"instructions"`
	Schemas      []measure    `json:"schemas"`
	AllSchemas   measure      `json:"allSchemas"`
	Deferred     measure      `json:"deferred"`
	Stands       []standRound `json:"stands"`
}

type standRound struct {
	Stand         string  `json:"stand"`
	Brief         measure `json:"brief"`
	Context       measure `json:"context"`
	Instructions  measure `json:"instructions"`
	Round         measure `json:"round"`
	DeferredRound measure `json:"deferredRound"`
}

type wireTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  *llm.Schema `json:"parameters"`
}

type wireGroup struct {
	Type        string `json:"type"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

func modes() []domainagent.Mode {
	return []domainagent.Mode{domainagent.ModeConfirm, domainagent.ModeAutonomous}
}

func measureAgent(ctx context.Context, c counter) (agentReport, error) {
	var measured agentReport
	for _, mode := range modes() {
		text, err := agentapp.Instructions(agentapp.SiteContext{Mode: string(mode)})
		if err != nil {
			return agentReport{}, err
		}
		row, err := c.measure(string(mode), text)
		if err != nil {
			return agentReport{}, err
		}
		measured.Instructions = append(measured.Instructions, row)
	}

	if err := measureSchemas(c, &measured); err != nil {
		return agentReport{}, err
	}
	for _, name := range sitemodeltest.Stands() {
		stand, err := measureStand(ctx, c, name, measured)
		if err != nil {
			return agentReport{}, err
		}
		measured.Stands = append(measured.Stands, stand)
	}
	return measured, nil
}

func measureSchemas(c counter, into *agentReport) error {
	built := tools.New(tools.Deps{}).Build(tools.Binding{Mode: domainagent.ModeAutonomous})
	schemas := make([]measure, 0, len(built))
	eager := make([]measure, 0, len(built))
	named := map[string]bool{}
	headers := []wireGroup{{Type: "tool_search"}}
	for i := range built {
		def := built[i].Def
		encoded, err := json.Marshal(wireTool{Name: def.Name, Description: def.Description, Parameters: def.Schema})
		if err != nil {
			return err
		}
		row, err := c.measure(def.Name, string(encoded))
		if err != nil {
			return err
		}
		schemas = append(schemas, row)
		group, onDemand := tools.OnDemand(def.Name)
		if !onDemand {
			eager = append(eager, row)
			continue
		}
		if !named[group.Name] {
			named[group.Name] = true
			headers = append(headers, wireGroup{Type: "namespace", Name: group.Name, Description: group.Description})
		}
	}
	encoded, err := json.Marshal(headers)
	if err != nil {
		return err
	}
	header, err := c.measure("group headers", string(encoded))
	if err != nil {
		return err
	}
	slices.SortStableFunc(schemas, func(a, b measure) int { return cmp.Or(b.Tokens-a.Tokens, cmp.Compare(a.Name, b.Name)) })
	into.Schemas = schemas
	into.AllSchemas = sum("every schema ("+strconv.Itoa(len(schemas))+" tools)", schemas...)
	into.Deferred = sum("deferred round ("+strconv.Itoa(len(eager))+" tools whole, "+strconv.Itoa(len(named))+" groups)",
		append(eager, header)...)
	return nil
}

func measureStand(ctx context.Context, c counter, name string, measured agentReport) (standRound, error) {
	model, err := sitemodeltest.Stand(name)
	if err != nil {
		return standRound{}, err
	}
	brief, err := model.Service().Brief(ctx, model.Site.ID)
	if err != nil {
		return standRound{}, err
	}
	briefRow, err := c.measure("brief", brief.Text())
	if err != nil {
		return standRound{}, err
	}

	situation := siteContextOf(model)
	situation.Mode = string(domainagent.ModeConfirm)
	text, err := agentapp.Situation(situation)
	if err != nil {
		return standRound{}, err
	}
	contextRow, err := c.measure("the turn's context message", text)
	if err != nil {
		return standRound{}, err
	}

	widest := measure{}
	for _, mode := range modes() {
		situation.Mode, situation.Brief = string(mode), brief.Text()
		text, err := agentapp.Instructions(situation)
		if err != nil {
			return standRound{}, err
		}
		row, err := c.measure("instructions with the brief, "+string(mode), text)
		if err != nil {
			return standRound{}, err
		}
		if row.Tokens > widest.Tokens {
			widest = row
		}
	}

	return standRound{
		Stand: name, Brief: briefRow, Context: contextRow, Instructions: widest,
		Round:         sum("every round, every tool", widest, measured.AllSchemas, contextRow),
		DeferredRound: sum("every round, tools on demand", widest, measured.Deferred, contextRow),
	}, nil
}

func siteContextOf(model *sitemodeltest.Memory) agentapp.SiteContext {
	situation := agentapp.SiteContext{SiteID: model.Site.ID, SiteName: model.Site.Name, Entities: len(model.Entities)}
	for i := range model.Pages {
		situation.Pages++
		if model.Pages[i].Status == pagemap.StatusPublished {
			situation.Published++
		}
		if model.Pages[i].EntityID == nil {
			situation.Unmapped++
		}
	}
	seeded := template.Seed()
	for i := range seeded {
		situation.Templates = append(situation.Templates, seeded[i].Name)
	}
	slices.Sort(situation.Templates)
	return situation
}

func (a agentReport) write(out io.Writer) error {
	return tables(out, func(s *sheet) {
		s.heading("THE AGENT, PER ROUND", "bytes", "tokens")
		for _, row := range a.Instructions {
			s.measured("instructions, "+row.Name, row)
		}
		s.measured(a.AllSchemas.Name, a.AllSchemas)
		s.measured(a.Deferred.Name, a.Deferred)
		for i := range a.Stands {
			stand := &a.Stands[i]
			for _, row := range []measure{stand.Brief, stand.Context, stand.Instructions, stand.Round, stand.DeferredRound} {
				s.measured(stand.Stand+": "+row.Name, row)
			}
		}
	}, func(s *sheet) {
		s.heading("EVERY TOOL SCHEMA, THE WIDEST FIRST", "bytes", "tokens")
		for _, row := range a.Schemas {
			s.measured(row.Name, row)
		}
	})
}
