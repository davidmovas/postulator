package pagetype

import (
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

type scopeKey struct {
	typeID string
	key    field.Key
}

type Set struct {
	types  []Type
	byID   map[string]Type
	scopes map[scopeKey]field.Scope
}

func NewSet(types []Type, scopes []field.Scope) Set {
	set := Set{types: types, byID: make(map[string]Type, len(types)), scopes: make(map[scopeKey]field.Scope, len(scopes))}
	for _, kind := range types {
		set.byID[kind.ID] = kind
	}
	for i := range scopes {
		set.scopes[scopeKey{typeID: scopes[i].PageTypeID, key: scopes[i].Key}] = scopes[i]
	}
	return set
}

func (s Set) Types() []Type {
	return s.types
}

func (s Set) ByID(id string) (Type, bool) {
	kind, found := s.byID[id]
	return kind, found
}

func (s Set) ByPreset(preset Preset) (Type, bool) {
	return ByPreset(s.types, preset)
}

func (s Set) ByRole(role Role) (Type, bool) {
	return ByRole(s.types, role)
}

func (s Set) TypeOf(page pagemap.Page) Type {
	if page.PageTypeID != nil {
		if kind, found := s.byID[*page.PageTypeID]; found && kind.SiteID == page.SiteID {
			return kind
		}
	}
	role, _, _ := Assign(Facts{WPType: page.WPType, Technical: page.TechnicalReason, Rendered: true})
	kind, _ := s.ByRole(role)
	return kind
}

func (s Set) NeedsEntity(page pagemap.Page) bool {
	return page.EntityID == nil && page.WPID != nil && page.Status != pagemap.StatusArchived &&
		!page.WPType.Term() && page.TechnicalReason == pagemap.TechnicalNone && s.TypeOf(page).Preset.Writable()
}

func (s Set) Scope(typeID string, key field.Key) (field.Scope, bool) {
	if key.Kind() != field.Meta && key.Instance() != "" {
		key = key.Kind()
	}
	if scope, stored := s.scopes[scopeKey{typeID: typeID, key: key}]; stored {
		return scope, true
	}
	kind, found := s.byID[typeID]
	if !found {
		return field.Scope{PageTypeID: typeID, Key: key, Links: field.NoLinks()}, false
	}
	scope := DefaultScope(kind.Preset, key, typeID)
	scope.Enabled = false
	return scope, false
}

func (s Set) Custom(typeID string) []field.Scope {
	out := make([]field.Scope, 0)
	for key := range s.scopes {
		if key.typeID == typeID && key.key.Kind() == field.Meta && s.scopes[key].Enabled {
			out = append(out, s.scopes[key])
		}
	}
	slices.SortFunc(out, func(a, b field.Scope) int { return strings.Compare(string(a.Key), string(b.Key)) })
	return out
}

func (s Set) Policy(page pagemap.Page) field.LinkPolicy {
	kind := s.TypeOf(page)
	scope, _ := s.Scope(kind.ID, LinkField(kind.Preset))
	if !scope.Enabled {
		return field.NoLinks()
	}
	return scope.Links
}
