package observe

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
)

type customRead struct {
	page   pagemap.Page
	keys   []string
	values map[string]string
}

func (s *Service) fields(ctx context.Context, state *State, readers Readers, batch int) (exhausted bool, err error) {
	if !readers.plugin(capabilityMeta) {
		return true, nil
	}
	defer func() {
		if exhausted || err != nil {
			s.unstage(state)
		}
	}()
	set, err := s.deps.Types.Set(ctx, state.SiteID)
	if err != nil {
		return false, err
	}
	held, err := s.staged(ctx, state, func(c context.Context) (*staged, error) {
		m, modelErr := s.model(c, state.SiteID)
		if modelErr != nil {
			return nil, modelErr
		}
		stored, storedErr := s.deps.Fields.ListBySite(c, state.SiteID)
		if storedErr != nil {
			return nil, storedErr
		}
		return &staged{model: m, stored: stored}, nil
	})
	if err != nil {
		return false, err
	}
	m, stored := held.model, held.stored

	reads := make([]customRead, 0, batch)
	exhausted = true
	for i := range m.pages {
		page := m.pages[i]
		if page.ID <= state.FieldsAfter || page.WPID == nil || page.Status == pagemap.StatusArchived || page.WPType.Term() {
			continue
		}
		if len(reads) == batch {
			exhausted = false
			break
		}
		state.FieldsAfter = page.ID
		if keys := wantedKeys(set, page, stored[page.ID], state.Full); len(keys) > 0 {
			reads = append(reads, customRead{page: page, keys: keys})
		}
	}
	for i := range reads {
		values, readErr := readers.Plugin.GetMeta(ctx, string(reads[i].page.WPType), *reads[i].page.WPID, reads[i].keys)
		if closedToUser(readErr) {
			reads[i].keys = nil
			continue
		}
		if readErr != nil {
			return false, readErr
		}
		reads[i].values = values
	}
	if len(reads) > 0 {
		if keepErr := s.keepCustom(ctx, state, readers, reads); keepErr != nil {
			return false, keepErr
		}
	}
	return exhausted, nil
}

func wantedKeys(set pagetype.Set, page pagemap.Page, stored []field.Value, full bool) []string {
	scopes := set.Custom(set.TypeOf(page).ID)
	held := make(map[field.Key]struct{}, len(stored))
	for i := range stored {
		held[stored[i].Key] = struct{}{}
	}
	keys := make([]string, 0, len(scopes))
	for i := range scopes {
		if _, read := held[scopes[i].Key]; full || !read {
			keys = append(keys, scopes[i].Key.Instance())
		}
	}
	return keys
}

func (s *Service) keepCustom(ctx context.Context, state *State, readers Readers, reads []customRead) error {
	tracked, err := s.findings(ctx, state.SiteID, finding.FieldChangedOnSite)
	if err != nil {
		return err
	}
	a := &absorber{service: s, state: state, readers: readers, tracked: tracked, pending: map[string]struct{}{}, now: s.now()}
	err = s.inUnit(ctx, func(c context.Context) error {
		for i := range reads {
			values := make([]field.Value, 0, len(reads[i].keys))
			for _, key := range reads[i].keys {
				values = append(values, field.NewValue(reads[i].page.ID, field.Instance(field.Meta, key), reads[i].values[key], a.now))
			}
			if keepErr := a.observeFields(c, reads[i].page, values); keepErr != nil {
				return keepErr
			}
			state.Counts.Custom += len(values)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if s.deps.Publisher != nil {
		return s.deps.Publisher.Publish(events.PagesChanged, events.PagesChangedPayload{SiteID: state.SiteID})
	}
	return nil
}
