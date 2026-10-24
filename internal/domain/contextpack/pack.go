package contextpack

import (
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/render"
)

type SectionName string

const (
	SectionPage      SectionName = "page"
	SectionRules     SectionName = "rules"
	SectionCard      SectionName = "card"
	SectionCurrent   SectionName = "current"
	SectionKeywords  SectionName = "keywords"
	SectionTrail     SectionName = "trail"
	SectionNotes     SectionName = "notes"
	SectionTargets   SectionName = "targets"
	SectionProducts  SectionName = "products"
	SectionSiblings  SectionName = "siblings"
	SectionChildren  SectionName = "children"
	SectionFields    SectionName = "fields"
	SectionGenerated SectionName = "generated"
)

type Section struct {
	Name     SectionName `json:"name"`
	Heading  string      `json:"heading"`
	Priority int         `json:"priority"`
	Lines    []string    `json:"lines"`
	Items    []string    `json:"items"`
	Tokens   int         `json:"tokens"`
}

type Pack struct {
	Field    field.Key `json:"field"`
	Sections []Section `json:"sections"`
	Tokens   int       `json:"tokens"`
}

var fieldNames = map[field.Key]string{
	field.Title: "Title", field.Excerpt: "Excerpt", field.Body: "Body", field.SEOTitle: "SEO title",
	field.SEODescription: "SEO description", field.ProductDescription: "Product description",
	field.ProductShortDescription: "Short description", field.VariationDescription: "Variation description",
	field.ImageAlt: "Image alt text", field.ImageTitle: "Image title",
}

var roles = map[pagetype.Preset]string{
	pagetype.Hub:           "a hub: it introduces its topic and links down to its children and sideways to related pages",
	pagetype.Entity:        "an article about one entity: it links up to its parents and sideways to related pages",
	pagetype.Product:       "a product page: its descriptions sell the product; it carries no children section",
	pagetype.StoreCategory: "a store category archive the store generates; it is read, never written",
	pagetype.Catalog:       "the shop or a listing root",
	pagetype.Technical:     "a technical page; it carries no links and is never written",
}

func FieldName(key field.Key) string {
	if name, known := fieldNames[key]; known {
		return name
	}
	if key.Kind() == field.Meta {
		return "Custom field " + key.Instance()
	}
	return string(key)
}

func Build(f Facts, scope field.Scope) Pack {
	sections := []Section{
		pageSection(f),
		rulesSection(f, scope),
		cardSection(f),
		currentSection(f, scope.Key),
		keywordsSection(f),
		trailSection(f),
		notesSection(f),
		targetsSection(f),
		productsSection(f),
		refsSection(SectionSiblings, "Siblings, the pages beside it", 5, f.Siblings),
		refsSection(SectionChildren, "Children, the pages below it", 5, f.Children),
		fieldsSection(f, scope.Key),
		generatedSection(f),
	}
	pack := Pack{Field: scope.Key, Sections: make([]Section, 0, len(sections))}
	for i := range sections {
		if len(sections[i].Lines) == 0 && len(sections[i].Items) == 0 {
			continue
		}
		sections[i].Tokens = sectionTokens(sections[i])
		pack.Tokens += sections[i].Tokens
		pack.Sections = append(pack.Sections, sections[i])
	}
	return pack
}

func sectionTokens(section Section) int {
	total := llm.Tokens(section.Heading)
	for _, line := range section.Lines {
		total += llm.Tokens(line)
	}
	for _, item := range section.Items {
		total += llm.Tokens(item)
	}
	return total
}

func pageSection(f Facts) Section {
	lines := []string{"Path: " + f.Page.Path, "Title: " + f.Page.Title}
	if f.H1 != f.Page.Title {
		lines = append(lines, "H1: "+f.H1)
	}
	kind := f.Page.Type
	if kind == "" {
		kind = "untyped"
	}
	if role := roles[f.Page.Preset]; role != "" {
		kind += ", " + role
	}
	lines = append(lines, "Page type: "+kind)
	state := "planned, not on the site yet"
	if f.Page.OnSite {
		state = "on the site, " + string(f.Status)
	}
	lines = append(lines, "WordPress type: "+string(f.WPType)+"; "+state)
	if f.Technical != pagemap.TechnicalNone {
		lines = append(lines, "Technical: "+string(f.Technical))
	}
	if f.ReadOnly != pagemap.ReadOnlyNone {
		lines = append(lines, "Read only: "+string(f.ReadOnly))
	}
	if terms := termsLine(f); terms != "" {
		lines = append(lines, "Terms: "+terms)
	}
	return Section{Name: SectionPage, Heading: "The page", Priority: 1, Lines: lines}
}

func termsLine(f Facts) string {
	parts := make([]string, 0, len(f.Terms))
	for _, term := range f.Terms {
		parts = append(parts, term.Taxonomy+" "+term.Name)
	}
	return strings.Join(parts, "; ")
}

func FieldRules(f Facts, scope field.Scope) []string {
	return rulesSection(f, scope).Lines
}

func rulesSection(f Facts, scope field.Scope) Section {
	lines := []string{"Field: " + FieldName(scope.Key)}
	if instruction := strings.TrimSpace(scope.Instruction); instruction != "" {
		lines = append(lines, "Instruction: "+instruction)
	}
	if length := lengthRule(scope); length != "" {
		lines = append(lines, "Length: "+length)
	}
	if current, held := f.Field(scope.Key); held && clientText(current) {
		if scope.Rewrites {
			lines = append(lines, "The current value may be rewritten.")
		} else {
			lines = append(lines, "The current value is the client's: keep it as it stands and write only what it lacks.")
		}
	}
	lines = append(lines, "Links: "+policyLine(scope.Links))
	return Section{Name: SectionRules, Heading: "The field's rules", Priority: 1, Lines: lines}
}

func clientText(row FieldFacts) bool {
	return strings.TrimSpace(row.Current) != "" && !row.Ours
}

func lengthRule(scope field.Scope) string {
	switch {
	case scope.MinLength > 0 && scope.MaxLength > 0:
		return "between " + strconv.Itoa(scope.MinLength) + " and " + strconv.Itoa(scope.MaxLength) + " characters"
	case scope.MinLength > 0:
		return "at least " + strconv.Itoa(scope.MinLength) + " characters"
	case scope.MaxLength > 0:
		return "at most " + strconv.Itoa(scope.MaxLength) + " characters"
	default:
		return ""
	}
}

func policyLine(policy field.LinkPolicy) string {
	if !policy.Links {
		return "none in this field"
	}
	rules := policy.Rules
	parts := make([]string, 0, 8)
	if rules.MaxLinks > 0 {
		parts = append(parts, "at most "+strconv.Itoa(rules.MaxLinks)+" links")
	}
	parts = append(parts, "at most "+strconv.Itoa(max(rules.MaxPerTarget, 1))+" per target")
	if rules.UpDepth > 0 {
		parts = append(parts, "up "+strconv.Itoa(rules.UpDepth)+" level(s)")
	}
	if rules.DownLinks {
		parts = append(parts, "down to the children")
	}
	parts = append(parts, "sideways to related pages weighted at least "+strconv.FormatFloat(rules.SiblingMinWeight, 'f', -1, 64))
	if rules.ParentLinkWithinParagraphs > 0 {
		parts = append(parts, "the parent linked within the first "+strconv.Itoa(rules.ParentLinkWithinParagraphs)+" paragraph(s)")
	}
	if rules.ChildrenSection {
		parts = append(parts, "a children section")
	}
	if policy.ForbidExternal {
		parts = append(parts, "no external links")
	}
	if policy.ForbidSelf {
		parts = append(parts, "no link to the page itself")
	}
	if anchors := anchorRules[policy.AnchorStrategy]; anchors != "" {
		parts = append(parts, anchors)
	}
	return strings.Join(parts, "; ")
}

var anchorRules = map[field.AnchorStrategy]string{
	field.AnchorPreferUser: "the client's anchors first",
	field.AnchorRotate:     "anchors rotated from one link to the next",
}

var parentSources = map[pagemap.ParentReason]string{
	pagemap.ParentJSONLD:   "the page's JSON-LD breadcrumb",
	pagemap.ParentARIA:     "the page's breadcrumb navigation",
	pagemap.ParentWP:       "its WordPress parent",
	pagemap.ParentURL:      "its address",
	pagemap.ParentWorkbook: "the workbook",
}

var relations = map[content.Relation]string{
	content.RelationUp:      "above it",
	content.RelationDown:    "below it",
	content.RelationSibling: "related to it",
}

var stockStates = map[string]string{
	"instock": "in stock", "outofstock": "out of stock", "onbackorder": "on backorder",
}

func cardSection(f Facts) Section {
	if f.Card == nil {
		return Section{Name: SectionCard}
	}
	lines := []string{"Entity: " + f.Card.Name}
	if definition := strings.TrimSpace(f.Card.Definition); definition != "" {
		lines = append(lines, "Definition: "+definition)
	}
	items := make([]string, 0, len(f.Card.Facts))
	for _, fact := range f.Card.Facts {
		items = append(items, "- "+fact.Label+": "+fact.Value)
	}
	return Section{Name: SectionCard, Heading: "The entity", Priority: 2, Lines: lines, Items: items}
}

func currentSection(f Facts, key field.Key) Section {
	row, held := f.Field(key)
	if !held || strings.TrimSpace(row.Current) == "" {
		return Section{Name: SectionCurrent, Heading: "The field's current value", Priority: 2, Lines: []string{"The field is empty."}}
	}
	owner := "the client's"
	if row.Ours {
		owner = "written by us before"
	}
	lines := []string{strings.TrimSpace(row.Current)}
	if key == field.Body || key == field.ProductDescription {
		lines = bodyLines(block.Parse(row.Current))
	}
	return Section{Name: SectionCurrent, Heading: "The field's current value (" + owner + ")", Priority: 2, Lines: lines}
}

func bodyLines(body block.Body) []string {
	out := make([]string, 0, len(body.Blocks))
	for i := range body.Blocks {
		held := &body.Blocks[i]
		if text := strings.TrimSpace(held.Text()); text != "" {
			out = append(out, text)
			continue
		}
		tags := body.ShortcodesAt(i)
		switch {
		case strings.TrimSpace(held.Raw) == "":
		case len(tags) > 0:
			out = append(out, "(a ["+tags[0]+"] shortcode, kept where it stands)")
		case held.Name != "":
			out = append(out, "(a "+held.Name+" block, kept where it stands)")
		default:
			out = append(out, "(markup with no text, kept where it stands)")
		}
	}
	return out
}

func keywordsSection(f Facts) Section {
	items := make([]string, 0, len(f.Keywords))
	for i, item := range f.Keywords {
		line := strconv.Itoa(i+1) + ". " + item.Text
		if item.Volume != nil {
			line += " (" + strconv.Itoa(*item.Volume) + " a month)"
		}
		items = append(items, line)
	}
	lines := make([]string, 0, 1)
	if f.Card != nil && len(f.Card.Anchors) > 0 {
		anchors := make([]string, 0, len(f.Card.Anchors))
		for _, anchor := range f.Card.Anchors {
			anchors = append(anchors, anchor.Text)
		}
		lines = append(lines, "Anchors others link here with: "+strings.Join(anchors, ", "))
	}
	return Section{Name: SectionKeywords, Heading: "Keywords, by volume", Priority: 3, Lines: lines, Items: items}
}

func trailSection(f Facts) Section {
	items := make([]string, 0, len(f.Trail))
	for _, ref := range f.Trail {
		items = append(items, refLine(ref))
	}
	lines := make([]string, 0, 1)
	if source := parentSources[f.ParentReason]; len(f.Trail) > 0 && source != "" {
		lines = append(lines, "The parent comes from "+source+".")
	}
	return Section{Name: SectionTrail, Heading: "The trail, nearest parent first", Priority: 3, Lines: lines, Items: items}
}

func notesSection(f Facts) Section {
	items := make([]string, 0, len(f.Notes))
	for _, item := range f.Notes {
		items = append(items, "- "+item.Label+": "+item.Text)
	}
	if f.Card != nil {
		for _, item := range f.Card.Notes {
			items = append(items, "- "+item.Label+" (the entity's): "+item.Text)
		}
	}
	return Section{Name: SectionNotes, Heading: "Notes", Priority: 3, Items: items}
}

func targetsSection(f Facts) Section {
	items := make([]string, 0, len(f.Targets))
	for i := range f.Targets {
		target := &f.Targets[i]
		line := "- " + target.Href + " (" + target.Title
		if relation := relations[target.Relation]; relation != "" {
			line += ", " + relation
		}
		if target.Required {
			line += ", required"
		}
		if !target.OnSite {
			line += ", not on the site yet"
		}
		line += ")"
		if len(target.Anchors) > 0 {
			line += ". Anchors: " + strings.Join(target.Anchors, " | ")
		}
		if target.Summary.Text != "" {
			line += ". About it: " + target.Summary.Text
		}
		items = append(items, line)
	}
	return Section{Name: SectionTargets, Heading: "Pages this field links to", Priority: 4, Items: items}
}

func productsSection(f Facts) Section {
	priority := 6
	if f.Page.Preset == pagetype.Hub || f.Page.Preset == pagetype.Catalog || f.Page.Preset == pagetype.StoreCategory {
		priority = 4
	}
	items := make([]string, 0, len(f.Products))
	for i := range f.Products {
		product := &f.Products[i]
		line := "- " + product.Title + " (" + product.Path + ")"
		if facts := productLine(product); facts != "" {
			line += ": " + facts
		}
		if product.Summary.Text != "" && product.Summary.Source != FromStructure {
			line += "; " + product.Summary.Text
		}
		items = append(items, line)
	}
	heading := "Products it lists"
	if f.ListedFrom == listedFromTwin && f.Twin != nil {
		heading = "Products of its store category " + f.Twin.Title
	}
	return Section{Name: SectionProducts, Heading: heading, Priority: priority, Items: items}
}

func productLine(product *Product) string {
	parts := make([]string, 0, 3)
	if product.Price != "" {
		parts = append(parts, "price "+product.Price)
	}
	if product.SKU != "" {
		parts = append(parts, "SKU "+product.SKU)
	}
	if stock, known := stockStates[product.Stock]; known {
		parts = append(parts, stock)
	} else if product.Stock != "" {
		parts = append(parts, product.Stock)
	}
	return strings.Join(parts, ", ")
}

func refsSection(name SectionName, heading string, priority int, refs []Ref) Section {
	items := make([]string, 0, len(refs))
	for _, ref := range refs {
		items = append(items, refLine(ref))
	}
	return Section{Name: name, Heading: heading, Priority: priority, Items: items}
}

func refLine(ref Ref) string {
	line := "- " + ref.Title + " (" + ref.Path
	if ref.Type != "" {
		line += ", " + ref.Type
	}
	line += ")"
	if ref.Summary.Text != "" {
		line += ": " + ref.Summary.Text
	}
	return line
}

func fieldsSection(f Facts, key field.Key) Section {
	items := make([]string, 0, len(f.Fields))
	for i := range f.Fields {
		row := &f.Fields[i]
		if row.Key == key || row.Key == field.Body || row.Key == field.ProductDescription || strings.TrimSpace(row.Current) == "" {
			continue
		}
		items = append(items, "- "+FieldName(row.Key)+": "+clip(clean(row.Current)))
	}
	return Section{Name: SectionFields, Heading: "Its other fields", Priority: 6, Items: items}
}

func generatedSection(f Facts) Section {
	if f.Root == "" {
		return Section{Name: SectionGenerated}
	}
	parts := make([]string, 0, len(f.Regions)+1)
	for _, region := range f.Regions {
		if text := regionText(region); text != "" {
			parts = append(parts, text)
		}
	}
	if f.Generated > 0 {
		parts = append(parts, strconv.Itoa(f.Generated)+" words the site generates inside the content area")
	}
	if len(parts) == 0 {
		return Section{Name: SectionGenerated}
	}
	return Section{
		Name: SectionGenerated, Heading: "What the site adds around the field", Priority: 6,
		Lines: []string{strings.Join(parts, "; ")},
	}
}

func regionText(region render.Region) string {
	where := "outside the content"
	if region.Inside {
		where = "inside the content area"
	}
	switch region.Kind {
	case render.RegionBreadcrumb:
		return "a breadcrumb " + where
	case render.RegionListing:
		return "a product listing of " + strconv.Itoa(region.Links) + " links " + where
	case render.RegionContentDynamic:
		return "content a shortcode or a dynamic block renders"
	default:
		return ""
	}
}
