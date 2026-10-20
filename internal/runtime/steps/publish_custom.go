package steps

import (
	"context"
	"slices"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/application/edit"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/run"
)

const (
	unitCustomStore  = "custom_store"
	unitCustomPlugin = "custom_plugin"
)

type customPlan struct {
	Key      field.Key      `json:"key"`
	Provider field.Provider `json:"provider"`
	Before   string         `json:"before"`
	Value    string         `json:"value"`
}

func customUnit(provider field.Provider) string {
	if provider == field.ProviderStore {
		return unitCustomStore
	}
	return unitCustomPlugin
}

func planCustom(ctx context.Context, deps Deps, sc *run.StepContext, client *wp.Client, itemType wp.ItemType, wpID int64,
	product *wp.Product) ([]customPlan, error) {
	answered, _, err := decodeArtifact[Fields](sc, run.ArtifactFields)
	if err != nil {
		return nil, err
	}
	set, err := deps.Types.Set(ctx, sc.Run.SiteID)
	if err != nil {
		return nil, err
	}
	kind := claimedType(set, sc)
	wanted := make([]customPlan, 0)
	pluginKeys := make([]string, 0)
	for _, value := range answered.Values {
		key := field.Key(value.Key)
		if key.Kind() != field.Meta {
			continue
		}
		scope, _ := set.Scope(kind.ID, key)
		if scope.Provider == field.ProviderStore && product == nil {
			continue
		}
		wanted = append(wanted, customPlan{Key: key, Provider: scope.Provider, Value: value.Value})
		if scope.Provider != field.ProviderStore {
			pluginKeys = append(pluginKeys, key.Instance())
		}
	}
	if len(wanted) == 0 {
		return wanted, nil
	}
	live, err := customValues(ctx, client, itemType, wpID, pluginKeys, product)
	if err != nil {
		return nil, err
	}
	keys := make([]field.Key, 0, len(wanted))
	for i := range wanted {
		wanted[i].Before = live[wanted[i].Key]
		keys = append(keys, wanted[i].Key)
	}
	kept, err := deps.keptFieldsOf(ctx, sc, wpID != 0, live, keys...)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(wanted, func(planned customPlan) bool {
		return kept[planned.Key] || planned.Before == planned.Value
	}), nil
}

func customValues(ctx context.Context, client *wp.Client, itemType wp.ItemType, wpID int64, pluginKeys []string,
	product *wp.Product) (map[field.Key]string, error) {
	live := make(map[field.Key]string)
	if wpID == 0 {
		return live, nil
	}
	if len(pluginKeys) > 0 {
		held, err := client.GetMeta(ctx, itemType, wpID, pluginKeys)
		if err != nil {
			return nil, err
		}
		for key, raw := range held {
			live[field.Instance(field.Meta, key)] = metaText(raw)
		}
	}
	if product != nil {
		for _, entry := range product.MetaData {
			live[field.Instance(field.Meta, entry.Key)] = metaText(entry.Value)
		}
	}
	return live, nil
}

func (p *publishPlan) customChanges() []field.Change {
	out := make([]field.Change, 0, len(p.Custom))
	for i := range p.Custom {
		out = append(out, field.NewChange(p.Custom[i].Key, p.Custom[i].Before, p.Custom[i].Value))
	}
	return out
}

func (p *publishPlan) customUnits() []string {
	units := make([]string, 0, 2)
	for i := range p.Custom {
		if unit := customUnit(p.Custom[i].Provider); !slices.Contains(units, unit) {
			units = append(units, unit)
		}
	}
	slices.Sort(units)
	return units
}

func checkCustom(ctx context.Context, client *wp.Client, itemType wp.ItemType, plan *publishPlan) ([]string, error) {
	moved := make([]string, 0)
	pluginKeys := make([]string, 0)
	store := false
	for i := range plan.Custom {
		if plan.Custom[i].Provider == field.ProviderStore {
			store = true
			continue
		}
		pluginKeys = append(pluginKeys, plan.Custom[i].Key.Instance())
	}
	var product *wp.Product
	if store {
		held, err := client.GetProduct(ctx, plan.WPID)
		if err != nil {
			return nil, err
		}
		product = &held
	}
	live, err := customValues(ctx, client, itemType, plan.WPID, pluginKeys, product)
	if err != nil {
		return nil, err
	}
	for i := range plan.Custom {
		if live[plan.Custom[i].Key] != plan.Custom[i].Before {
			moved = append(moved, string(plan.Custom[i].Key))
		}
	}
	return moved, nil
}

func writeCustom(ctx context.Context, sc *run.StepContext, client *wp.Client, itemType wp.ItemType, state *publishState,
	unit string) (run.Result, error) {
	plan := state.Plan
	planned := make([]customPlan, 0, len(plan.Custom))
	writes := make([]edit.Write, 0, len(plan.Custom))
	for i := range plan.Custom {
		if customUnit(plan.Custom[i].Provider) != unit {
			continue
		}
		planned = append(planned, plan.Custom[i])
		writes = append(writes, edit.Write{
			Key: plan.Custom[i].Key, Value: &plan.Custom[i].Value, Expected: expectedOf(plan.Custom[i].Before),
			Provider: plan.Custom[i].Provider,
		})
	}
	target := edit.Target{Type: string(itemType), ID: state.Result.WPID, Product: plan.Product}
	results, err := editorFor(client).Write(ctx, target, writes)
	if err != nil {
		return state.failed(err)
	}
	if moved := movedKeys(results); len(moved) > 0 {
		return state.stopped(needsHuman(fieldsMoved(sc.Page.Path, moved)))
	}
	for i := range planned {
		written, _ := writtenValue(results, planned[i].Key)
		state.Result.PreviousFields = append(state.Result.PreviousFields,
			FieldWrite{Key: planned[i].Key, Before: planned[i].Before, Written: written, Provider: planned[i].Provider})
	}
	state.wrote(unit)
	return state.wait("wrote the custom fields of " + sc.Page.Path)
}

func recordCustom(ctx context.Context, deps Deps, sc *run.StepContext, state *publishState) error {
	for _, written := range state.Result.PreviousFields {
		if err := recordField(ctx, deps, sc, sc.Page.ID, written.Key, written.Written); err != nil {
			return err
		}
	}
	return nil
}

func putTheCustomBack(ctx context.Context, deps Deps, sc *run.StepContext, work revertWork) (string, bool) {
	if len(work.published.PreviousFields) == 0 {
		return "", true
	}
	itemType := onSiteType(work.page)
	writes := make(map[string]wp.MetaWrite)
	store := make(map[string]string)
	for _, written := range work.published.PreviousFields {
		if written.Provider == field.ProviderStore {
			store[written.Key.Instance()] = written.Before
			continue
		}
		hash := block.BodyHash(written.Written)
		var before *string
		if written.Before != "" {
			before = &written.Before
		}
		writes[written.Key.Instance()] = wp.MetaWrite{Value: before, Expected: &hash}
	}
	if len(store) > 0 {
		held, err := work.client.GetProduct(ctx, work.published.WPID)
		if err != nil {
			return storeReason(err), false
		}
		for _, written := range work.published.PreviousFields {
			if written.Provider != field.ProviderStore {
				continue
			}
			current := ""
			for _, entry := range held.MetaData {
				if entry.Key == written.Key.Instance() {
					current = metaText(entry.Value)
				}
			}
			if current != written.Written {
				return ReasonRevertCustomEdited, false
			}
		}
		if _, err = work.client.UpdateProduct(ctx, work.published.WPID, wp.UpdateProduct{MetaData: store}); err != nil {
			return storeReason(err), false
		}
	}
	if len(writes) > 0 {
		if _, err := work.client.PutMeta(ctx, itemType, work.published.WPID, writes); err != nil {
			if _, conflict := movedOf(err); conflict {
				return ReasonRevertCustomEdited, false
			}
			return err.Error(), false
		}
	}
	for _, written := range work.published.PreviousFields {
		if err := recordField(ctx, deps, sc, work.page.ID, written.Key, written.Before); err != nil {
			return err.Error(), false
		}
	}
	return "", true
}
