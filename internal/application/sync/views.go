package sync

import "github.com/davidmovas/postulator/internal/domain/site"

type Plugin struct {
	Version         string   `json:"version"`
	RequiredVersion string   `json:"requiredVersion"`
	SEOPlugin       string   `json:"seoPlugin"`
	Capabilities    []string `json:"capabilities"`
	Installed       bool     `json:"installed"`
	Outdated        bool     `json:"outdated"`
}

func pluginView(found site.Extensions) Plugin {
	capabilities := found.Plugin.Capabilities
	if capabilities == nil {
		capabilities = []string{}
	}
	return Plugin{
		Version:         found.Plugin.Version,
		RequiredVersion: found.PluginRequired,
		SEOPlugin:       found.Plugin.SEOPlugin,
		Capabilities:    capabilities,
		Installed:       found.Plugin.Installed,
		Outdated:        found.PluginOutdated,
	}
}
