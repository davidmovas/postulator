package contextpack

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	BriefMaxBytes = 4096
	hubTitleRunes = 48
)

type LinkHealth struct {
	Audited         int `json:"audited"`
	Targets         int `json:"targets"`
	Satisfied       int `json:"satisfied"`
	Missing         int `json:"missing"`
	MissingRequired int `json:"missingRequired"`
	Pending         int `json:"pending"`
	Orphans         int `json:"orphans"`
	OffGraph        int `json:"offGraph"`
}

type Brief struct {
	Name     string     `json:"name"`
	BaseURL  string     `json:"baseUrl"`
	Overview Overview   `json:"overview"`
	Links    LinkHealth `json:"links"`
}

type shape struct {
	hubs, children, types, codes int
}

var shapes = []shape{
	{hubs: 6, children: 4, types: 10, codes: 8},
	{hubs: 6, children: 2, types: 10, codes: 8},
	{hubs: 6, children: 0, types: 10, codes: 8},
	{hubs: 3, children: 0, types: 6, codes: 5},
	{hubs: 0, children: 0, types: 4, codes: 3},
}

func (b Brief) Text() string {
	if b.Overview.Pages == 0 {
		return "THE SITE\n" + b.Name + " (" + b.BaseURL + "): no page observed yet; sync the site before anything else.\n"
	}
	text := ""
	for _, fit := range shapes {
		text = b.render(fit)
		if len(text) <= BriefMaxBytes {
			return text
		}
	}
	return text
}

func (b Brief) render(fit shape) string {
	var out strings.Builder
	o := b.Overview
	out.WriteString("THE SITE\n")
	out.WriteString(b.Name + " (" + b.BaseURL + "): " + countOf(o.Pages, "page", "pages") + ", " + presence(o) + "; " +
		unmapped(o.Unmapped))
	if o.OnSite > 0 {
		out.WriteString("; " + empty(o.Empty))
	}
	out.WriteString(".\n")

	out.WriteString("\nPAGE TYPES\n")
	for i, kind := range o.Types {
		if i == fit.types {
			out.WriteString("- and " + strconv.Itoa(len(o.Types)-i) + " more types\n")
			break
		}
		name := kind.Name
		if name == "" {
			name = "No type"
		}
		out.WriteString("- " + name + ": " + countOf(kind.Pages, "page", "pages"))
		if kind.Planned > 0 {
			out.WriteString(", " + strconv.Itoa(kind.Planned) + " of them planned")
		}
		out.WriteString("\n")
	}

	out.WriteString("\nHUBS\n")
	b.writeHubs(&out, fit)

	out.WriteString("\nLINKS\n")
	out.WriteString(b.Links.text())

	out.WriteString("\nOPEN FINDINGS\n")
	if len(o.Findings) == 0 {
		out.WriteString("none\n")
	}
	for i, code := range o.Findings {
		if i == fit.codes {
			out.WriteString("- and " + strconv.Itoa(len(o.Findings)-i) + " more codes\n")
			break
		}
		out.WriteString("- " + string(code.Code) + ": " + strconv.Itoa(code.Count) + "\n")
	}
	return out.String()
}

func (b Brief) writeHubs(out *strings.Builder, fit shape) {
	hubs := b.Overview.Hubs
	if len(hubs) == 0 {
		out.WriteString("No page is typed as a hub or a catalog yet.\n")
		return
	}
	out.WriteString("The top two levels of the hub tree, with the pages under each hub:\n")
	for i := range hubs {
		if i == fit.hubs {
			out.WriteString("- and " + strconv.Itoa(len(hubs)-i) + " more hubs\n")
			return
		}
		out.WriteString("- " + hubLine(hubs[i]) + "\n")
		for j := range hubs[i].Children {
			if j == fit.children {
				out.WriteString("  - and " + strconv.Itoa(len(hubs[i].Children)-j) + " more\n")
				break
			}
			out.WriteString("  - " + hubLine(hubs[i].Children[j]) + "\n")
		}
	}
}

func hubLine(hub Hub) string {
	under := "none"
	if hub.Pages > 0 {
		under = countOf(hub.Pages, "page", "pages")
	}
	title := hub.Title
	if utf8.RuneCountInString(title) > hubTitleRunes {
		title = strings.TrimSpace(string([]rune(title)[:hubTitleRunes])) + "…"
	}
	return hub.Path + " " + title + ": " + under
}

func (l LinkHealth) text() string {
	if l.Audited == 0 {
		return "No page is mapped to an entity, so the graph owes no links yet.\n"
	}
	var out strings.Builder
	out.WriteString("Of " + countOf(l.Audited, "audited page", "audited pages") + " the graph owes " + countOf(l.Targets, "link", "links") +
		": " + strconv.Itoa(l.Satisfied) + " placed, " + strconv.Itoa(l.Missing) + " missing")
	if l.MissingRequired > 0 {
		out.WriteString(" (" + strconv.Itoa(l.MissingRequired) + " of them required)")
	}
	if l.Pending > 0 {
		out.WriteString(", " + strconv.Itoa(l.Pending) + " waiting on a page")
	}
	out.WriteString(". ")
	switch l.Orphans {
	case 0:
		out.WriteString("No page is an orphan")
	case 1:
		out.WriteString("1 page is an orphan")
	default:
		out.WriteString(strconv.Itoa(l.Orphans) + " pages are orphans")
	}
	switch l.OffGraph {
	case 0:
		out.WriteString("; every link stays on the graph.\n")
	case 1:
		out.WriteString("; 1 link points off the graph.\n")
	default:
		out.WriteString("; " + strconv.Itoa(l.OffGraph) + " links point off the graph.\n")
	}
	return out.String()
}

func presence(o Overview) string {
	switch {
	case o.Planned == 0:
		return "all on the site"
	case o.OnSite == 0:
		return "all planned"
	default:
		return strconv.Itoa(o.OnSite) + " on the site and " + strconv.Itoa(o.Planned) + " planned"
	}
}

func unmapped(count int) string {
	switch count {
	case 0:
		return "every page carries an entity"
	case 1:
		return "1 carries no entity"
	default:
		return strconv.Itoa(count) + " carry no entity"
	}
}

func empty(count int) string {
	switch count {
	case 0:
		return "every page on the site holds text"
	case 1:
		return "1 on the site holds no text yet"
	default:
		return strconv.Itoa(count) + " on the site hold no text yet"
	}
}

func countOf(count int, one, many string) string {
	if count == 1 {
		return "1 " + one
	}
	return strconv.Itoa(count) + " " + many
}
