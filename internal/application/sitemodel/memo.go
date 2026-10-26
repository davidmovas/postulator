package sitemodel

import (
	"context"
	"sync"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
)

type memoKey struct{}

type pictureKey struct {
	pageID string
	field  string
	budget int
}

type remembered struct {
	picture Picture
	err     error
}

type memo struct {
	mu       sync.Mutex
	bases    map[string]contextpack.Snapshot
	pictures map[pictureKey]remembered
}

func (s *Service) Remembering(ctx context.Context) context.Context {
	if _, held := ctx.Value(memoKey{}).(*memo); held {
		return ctx
	}
	return context.WithValue(ctx, memoKey{}, &memo{bases: map[string]contextpack.Snapshot{}, pictures: map[pictureKey]remembered{}})
}

func memoOf(ctx context.Context) *memo {
	if held, ok := ctx.Value(memoKey{}).(*memo); ok {
		return held
	}
	return nil
}

func (m *memo) picture(req PackRequest) (remembered, bool) {
	if m == nil || req.Claim != nil || req.directed() {
		return remembered{}, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	held, found := m.pictures[pictureKey{pageID: req.PageID, field: req.Field, budget: req.Budget}]
	return held, found
}

func (m *memo) keep(req PackRequest, picture Picture, err error) {
	if m == nil || req.Claim != nil || req.directed() {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pictures[pictureKey{pageID: req.PageID, field: req.Field, budget: req.Budget}] = remembered{picture: picture, err: err}
}

func (m *memo) base(siteID string) (contextpack.Snapshot, bool) {
	if m == nil {
		return contextpack.Snapshot{}, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	held, found := m.bases[siteID]
	return fresh(held), found
}

func (m *memo) keepBase(siteID string, snapshot contextpack.Snapshot) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bases[siteID] = snapshot
}

func fresh(snapshot contextpack.Snapshot) contextpack.Snapshot {
	snapshot.Bodies, snapshot.Written = map[string]block.Snapshot{}, map[string][]string{}
	snapshot.Renders, snapshot.Links, snapshot.Findings = map[string]render.Record{}, []pagemap.PageLink{}, []finding.Finding{}
	return snapshot
}
