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
	{Name: "graph", Description: "The entity graph: entities, keywords, anchors, parent and related edges, the page tree's parents, " +
		"proposals from pages or keywords, authority scores."},
	{Name: "pages", Description: "The pages: find them, walk the tree, read one whole with its blocks and fields; plan, " +
		"change, map, retype or delete pages, write a field or a block, grade a page."},
	{Name: "links", Description: "The contextual links: a page's or a section's, what the graph owes, the orphans and what to add."},
	{Name: "pagetypes", Description: "Page types: the client's types over presets, their field scopes and link policies, " +
		"and the rules that classify pages, previewed before they apply."},
	{Name: "templates", Description: "Content templates, and their overrides for one site or one page."},
	{Name: "runs", Description: "Runs that write, relink, audit, repair or revert pages: start, pause, resume, " +
		"cancel, retry, regenerate and revert them, and read their items, events and artifacts."},
	{Name: "sync", Description: "Pull the live WordPress content into the page map and probe the companion plugin."},
	{Name: "reports", Description: "The site's totals."},
	{Name: "imports", Description: "Spreadsheet import and export, and saved column mappings."},
	{Name: "findings", Description: "What the model found that disagrees with the site or the workbook, and its resolution."},
	{Name: "models", Description: "Language models: the catalog, role profiles, provider keys, probes and spend."},
	{Name: "schedules", Description: "Schedules that start runs on a cron expression or an interval."},
	{Name: "plans", Description: "Plans: an ordered set of writes put in front of the person as one card, run once approved."},
}

var orienting = []string{
	"sites_list", "sites_get", "reports_site_overview", "graph_read", "pages_find", "pages_tree",
	"pages_get", "links", "runs", "templates_list",
}

func OnDemand(name string) (Group, bool) {
	if slices.Contains(orienting, name) {
		return Group{}, false
	}
	return groupOf(name)
}

func groupOf(name string) (Group, bool) {
	prefix, _, _ := strings.Cut(name, "_")
	at := slices.IndexFunc(groups, func(group Group) bool { return group.Name == prefix })
	if at < 0 {
		return Group{}, false
	}
	return groups[at], true
}
