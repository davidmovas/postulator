package observe

import (
	"context"
	"sync"
	"time"

	"github.com/davidmovas/postulator/internal/domain/field"
)

type stageKey struct {
	siteID  string
	started time.Time
	stage   Stage
}

type staged struct {
	model    siteModel
	observed map[string]time.Time
	stored   map[string][]field.Value
}

type stages struct {
	mu   sync.Mutex
	held map[stageKey]*staged
}

func keyOfStage(state *State) stageKey {
	return stageKey{siteID: state.SiteID, started: state.StartedAt, stage: state.Stage}
}

func (s *Service) staged(ctx context.Context, state *State, load func(context.Context) (*staged, error)) (*staged, error) {
	key := keyOfStage(state)
	s.stages.mu.Lock()
	held, found := s.stages.held[key]
	s.stages.mu.Unlock()
	if found {
		return held, nil
	}
	loaded, err := load(ctx)
	if err != nil {
		return nil, err
	}
	s.stages.mu.Lock()
	defer s.stages.mu.Unlock()
	for other := range s.stages.held {
		if other.siteID == key.siteID {
			delete(s.stages.held, other)
		}
	}
	s.stages.held[key] = loaded
	return loaded, nil
}

func (s *Service) unstage(state *State) {
	s.stages.mu.Lock()
	defer s.stages.mu.Unlock()
	delete(s.stages.held, keyOfStage(state))
}
