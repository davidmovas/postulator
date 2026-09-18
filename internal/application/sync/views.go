package sync

import "github.com/davidmovas/postulator/internal/domain/site"

type Plugin struct {
	Version      string   `json:"version"`
	SEOPlugin    string   `json:"seoPlugin"`
	Capabilities []string `json:"capabilities"`
	Installed    bool     `json:"installed"`
}

func pluginView(state site.PluginState) Plugin {
	capabilities := state.Capabilities
	if capabilities == nil {
		capabilities = []string{}
	}
	return Plugin{
		Version:      state.Version,
		SEOPlugin:    state.SEOPlugin,
		Capabilities: capabilities,
		Installed:    state.Installed,
	}
}
