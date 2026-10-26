package app

import (
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/application/support"
)

func (c *Core) buildSupport(stores repos, writing authoring, running runtimeParts) *support.Service {
	return support.New(support.Deps{
		Version: Version, PluginRequired: wp.PluginMinimum, Sites: stores.sites, Pages: stores.pages, Renders: stores.renders,
		Types: writing.pageTypes, Entities: stores.entities, Edges: stores.edges, Findings: stores.findings,
		Observation: running.observe, Conversations: stores.conversations, Calls: stores.toolCalls, Messages: stores.messages,
		Clock: stores.now,
	})
}
