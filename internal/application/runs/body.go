package runs

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
)

const pathsNamed = 3

type bodyless struct {
	types   []string
	paths   []string
	planned []string
	fields  []field.Key
}

func (s *Service) bodySwitch(ctx context.Context, siteID string, targets []string, recipe []template.StepSpec) ([]string, error) {
	set, err := s.types.Set(ctx, siteID)
	if err != nil {
		return nil, err
	}
	var silent bodyless
	writing := make([]string, 0)
	for _, pageID := range targets {
		page, getErr := s.pages.Get(ctx, pageID)
		if getErr != nil {
			return nil, getErr
		}
		kind := set.TypeOf(page)
		written := writtenBy(set, kind.ID)
		if kind.ID == "" || run.WritesBody(written) {
			writing = append(writing, page.Path)
			continue
		}
		if !slices.Contains(silent.types, kind.Name) {
			silent.types = append(silent.types, kind.Name)
		}
		silent.paths = append(silent.paths, page.Path)
		if page.WPID == nil {
			silent.planned = append(silent.planned, page.Path)
		}
		for _, key := range written {
			if !slices.Contains(silent.fields, key) {
				silent.fields = append(silent.fields, key)
			}
		}
	}
	if len(silent.paths) == 0 {
		return nil, nil
	}
	types := joined(silent.types)
	if len(writing) > 0 {
		return nil, invalid("these pages cannot share a run: "+types+" pages write no body ("+named(silent.paths)+
			") and the others do ("+named(writing)+"); start a run for each", "pageIds").
			WithDetail("bodyless", silent.paths).WithDetail("writing", writing)
	}
	if len(silent.planned) > 0 {
		return nil, invalid(types+" pages write no body, so a run cannot create "+named(silent.planned)+
			", which is not on the site yet; switch the body on for "+types+" in Page types, or create the page in WordPress first",
			"pageIds").WithDetail("paths", silent.planned)
	}
	fields := produced(recipe, silent.fields)
	if len(fields) == 0 {
		return nil, invalid(types+" pages write no field the template's steps make, so the run has nothing to write; "+
			"switch a field on for "+types+" in Page types", "pageIds")
	}
	return fields, nil
}

func writtenBy(set pagetype.Set, typeID string) []field.Key {
	out := make([]field.Key, 0)
	if typeID == "" {
		return out
	}
	for _, spec := range field.Catalog() {
		if spec.Instanced || !spec.AIWritable() {
			continue
		}
		if scope, _ := set.Scope(typeID, spec.Key); scope.Enabled && scope.AIWrites {
			out = append(out, spec.Key)
		}
	}
	custom := set.Custom(typeID)
	for i := range custom {
		if custom[i].AIWrites {
			out = append(out, custom[i].Key)
		}
	}
	return out
}

func produced(recipe []template.StepSpec, keys []field.Key) []string {
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		producer := string(run.Producer(key, keys))
		if slices.ContainsFunc(recipe, func(step template.StepSpec) bool { return step.Enabled && step.Name == producer }) {
			out = append(out, string(key))
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

func named(paths []string) string {
	shown := paths[:min(len(paths), pathsNamed)]
	said := strings.Join(shown, ", ")
	if more := len(paths) - len(shown); more > 0 {
		said += " and " + strconv.Itoa(more) + " more"
	}
	return said
}

func joined(names []string) string {
	if len(names) < 2 {
		return strings.Join(names, "")
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}
