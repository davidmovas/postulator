package app

import (
	"time"

	"github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/tools"
	agentrunner "github.com/davidmovas/postulator/internal/transport/agent"
)

type agentParts struct {
	tools *tools.Registry
	agent *agent.Service
}

func (c *Core) buildAgent(stores repos, llm llmParts, writing authoring, services useCases) agentParts {
	values := stores.values
	toolRegistry := tools.New(tools.Deps{
		Sites: services.sites, Graph: services.graph, Pages: services.pages, Templates: writing.templates,
		Runs: services.runs, Sync: services.sync, Reports: services.reports, Imports: services.imports,
		Models: services.models, Content: writing.content, Schedules: services.schedules,
		Actions: stores.actions, Publisher: c.Events, Clock: stores.now,
	})

	return agentParts{
		tools: toolRegistry,
		agent: agent.New(agent.Deps{
			Conversations:     stores.conversations,
			Messages:          stores.messages,
			Actions:           stores.actions,
			Calls:             stores.toolCalls,
			Sites:             stores.sites,
			Reports:           services.reports,
			Templates:         writing.templates,
			Profiles:          llm.profiles,
			LLM:               llm.client,
			Registry:          toolRegistry,
			Runner:            c.agentRunner(stores, llm, toolRegistry),
			Publisher:         c.Events,
			Clock:             stores.now,
			TurnTimeout:       func() time.Duration { return agent.TurnTimeout(values) },
			LoopLimit:         func() int { return agent.LoopLimit(values) },
			HistoryBudget:     func() int { return agent.HistoryBudgetChars(values) },
			MaxToolResult:     func() int { return agent.MaxToolResultBytes(values) },
			HistoryToolResult: func() int { return agent.HistoryToolResultBytes(values) },
		}),
	}
}

func (c *Core) agentRunner(stores repos, llm llmParts, toolRegistry *tools.Registry) *agentrunner.Runner {
	values := stores.values
	return agentrunner.New(agentrunner.Deps{
		Client:      llm.client,
		Registry:    toolRegistry,
		History:     stores.history,
		Catalog:     llm.catalog,
		Clock:       stores.now,
		Logger:      c.logger,
		ToolLoading: func() agent.ToolLoading { return agent.ToolLoadingOf(values) },
	})
}
