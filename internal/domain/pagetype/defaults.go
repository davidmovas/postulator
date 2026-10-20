package pagetype

import "github.com/davidmovas/postulator/internal/domain/field"

var defaultsOn = map[Preset][]field.Key{
	Hub:     {field.Body, field.SEOTitle, field.SEODescription},
	Entity:  {field.Body, field.SEOTitle, field.SEODescription},
	Product: {field.ProductDescription, field.ProductShortDescription, field.SEOTitle, field.SEODescription},
}

var presetRules = map[Preset]field.LinkRules{
	Hub:           {UpDepth: 1, DownLinks: true, SiblingMinWeight: 0.5, MaxLinks: 20, MaxPerTarget: 1, ParentLinkWithinParagraphs: 2, ChildrenSection: true},
	Entity:        {UpDepth: 2, DownLinks: true, SiblingMinWeight: 0.5, MaxLinks: 12, MaxPerTarget: 1, ParentLinkWithinParagraphs: 2},
	StoreCategory: {UpDepth: 1, DownLinks: true, SiblingMinWeight: 0.5, MaxLinks: 15, MaxPerTarget: 1, ParentLinkWithinParagraphs: 1, ChildrenSection: true},
	Catalog:       {UpDepth: 1, DownLinks: true, SiblingMinWeight: 0.5, MaxLinks: 15, MaxPerTarget: 1, ParentLinkWithinParagraphs: 1, ChildrenSection: true},
	Product:       {UpDepth: 1, SiblingMinWeight: 1, MaxLinks: 3, MaxPerTarget: 1, ParentLinkWithinParagraphs: 1},
}

func LinkField(preset Preset) field.Key {
	if preset == Product {
		return field.ProductDescription
	}
	return field.Body
}

func DefaultScope(preset Preset, key field.Key, typeID string) field.Scope {
	spec, known := field.Lookup(key)
	scope := field.Scope{PageTypeID: typeID, Key: key, AIWrites: !known || spec.AIWritable(), Links: field.NoLinks()}
	for _, on := range defaultsOn[preset] {
		if on == key {
			scope.Enabled = true
		}
	}
	if rules, linked := presetRules[preset]; linked && key == LinkField(preset) {
		scope.Links = field.LinkPolicy{Links: true, Rules: rules, ForbidExternal: true, ForbidSelf: true, AnchorStrategy: field.AnchorPreferUser}
	}
	return scope
}

func DefaultScopes(preset Preset, typeID string) []field.Scope {
	keys := defaultsOn[preset]
	out := make([]field.Scope, 0, len(keys))
	for _, key := range keys {
		out = append(out, DefaultScope(preset, key, typeID))
	}
	return out
}
