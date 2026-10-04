package tools

import (
	"slices"
	"strings"
)

type Group struct {
	Name        string
	Description string
}

var groups = []Group{
	{Name: "sites", Description: "WordPress sites: register, change or delete one and set its defaults."},
	{Name: "graph", Description: "The entity graph: entities, keywords, anchors, parent and related edges, " +
		"proposals from pages or keywords, authority scores."},
	{Name: "pages", Description: "The page map: plan, change or delete pages, map them to entities, " +
		"set canonicals, replace recorded links, preview a page."},
	{Name: "templates", Description: "Content templates, and their overrides for one site or one page."},
	{Name: "policies", Description: "Link policies: the internal links a page may carry and owes."},
	{Name: "runs", Description: "Runs that write, relink, audit, repair or revert pages: start, pause, resume, " +
		"cancel, retry, regenerate and revert them, and read their items, events and artifacts."},
	{Name: "sync", Description: "Pull the live WordPress content into the page map and probe the companion plugin."},
	{Name: "reports", Description: "Reports on a page or a run, and link audits."},
	{Name: "imports", Description: "Spreadsheet import and export, and saved column mappings."},
	{Name: "models", Description: "Language models: the catalog, role profiles, provider keys, probes and spend."},
	{Name: "content", Description: "Grade a live page against its template and graph."},
	{Name: "schedules", Description: "Schedules that start runs on a cron expression or an interval."},
}

var orienting = []string{
	"sites_list", "sites_get", "reports_site_overview", "graph_list_entities", "pages_list", "pages_get",
	"pages_tree", "runs_list", "runs_get", "templates_list",
}

func OnDemand(name string) (Group, bool) {
	if slices.Contains(orienting, name) {
		return Group{}, false
	}
	return groupOf(name)
}

func groupOf(name string) (Group, bool) {
	prefix, _, named := strings.Cut(name, "_")
	if !named {
		return Group{}, false
	}
	at := slices.IndexFunc(groups, func(group Group) bool { return group.Name == prefix })
	if at < 0 {
		return Group{}, false
	}
	return groups[at], true
}
