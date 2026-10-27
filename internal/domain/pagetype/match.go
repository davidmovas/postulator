package pagetype

import (
	"strings"
	"unicode"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

var synonyms = map[string]Preset{
	"hub": Hub, "hubs": Hub, "commercial taxonomy": Hub, "taxonomy": Hub,
	"root": Hub, "root entity": Hub, "pillar": Hub, "store category": StoreCategory, "product category": StoreCategory,
	"category": Category, "categories": Category, "landing": Landing, "landing page": Landing,
	"entity": Entity, "entities": Entity, "topic": Entity, "custom": Entity, "geo": Entity, "geographic": Entity,
	"geographic entity": Entity, "guide": Guide, "guides": Guide, "comparison": Guide, "article": Guide,
	"info": Info, "information": Info, "informational": Info,
	"product": Product, "products": Product, "compound": Product, "compound product": Product,
	"product compound": Product, "product owner": Product,
	"catalog": Catalog, "shop": Catalog, "technical": Technical,
}

func Match(cell string, types []Type, wpType pagemap.WPType) (Type, bool) {
	if wpType.Term() {
		return ByRole(types, RoleArchive)
	}
	key := graph.Key(cell)
	if key == "" {
		return Type{}, false
	}
	for i := range types {
		if graph.Key(types[i].Name) == key {
			return onPage(types, types[i])
		}
	}
	preset, known := synonyms[words(cell)]
	if !known {
		return Type{}, false
	}
	found, ok := ByPreset(types, preset)
	if !ok {
		found, ok = ByRole(types, preset.Role())
	}
	if !ok {
		return Type{}, false
	}
	return onPage(types, found)
}

func onPage(types []Type, named Type) (Type, bool) {
	if named.Preset.Role() == RoleArchive {
		return ByRole(types, RoleHub)
	}
	return named, true
}

func ByRole(types []Type, role Role) (Type, bool) {
	for _, preset := range rolePresets[role] {
		if kind, found := ByPreset(types, preset); found {
			return kind, true
		}
	}
	return Type{}, false
}

func ByPreset(types []Type, preset Preset) (Type, bool) {
	var oldest *Type
	for i := range types {
		if types[i].Preset != preset {
			continue
		}
		if oldest == nil || types[i].CreatedAt.Before(oldest.CreatedAt) || types[i].CreatedAt.Equal(oldest.CreatedAt) && types[i].ID < oldest.ID {
			oldest = &types[i]
		}
	}
	if oldest == nil {
		return Type{}, false
	}
	return *oldest, true
}

func sameName(a, b string) bool {
	return graph.Key(a) == graph.Key(b)
}

func words(text string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}), " ")
}
