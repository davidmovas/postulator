package profiles

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type profileStore interface {
	List(ctx context.Context) (map[llm.Role]llm.ModelRef, error)
	Set(ctx context.Context, role llm.Role, ref llm.ModelRef, at time.Time) error
	Delete(ctx context.Context, role llm.Role) error
}

type siteReader interface {
	Get(ctx context.Context, id string) (site.Site, error)
}

type roleDefaults interface {
	Default(role llm.Role) (llm.ModelRef, error)
}

type Profiles struct {
	store    profileStore
	sites    siteReader
	defaults roleDefaults
	clock    clock.Clock
}

func New(store profileStore, sites siteReader, defaults roleDefaults, clk clock.Clock) *Profiles {
	return &Profiles{store: store, sites: sites, defaults: defaults, clock: clk}
}

func (p *Profiles) Resolve(ctx context.Context, siteID string, role llm.Role, templateProfiles map[llm.Role]llm.ModelRef) (llm.ModelRef, error) {
	if !role.Valid() {
		return llm.ModelRef{}, errors.New(errors.Invalid, "a model profile role must be one of the known roles").
			WithDetail("role", string(role))
	}

	if ref, ok := pick(templateProfiles, role); ok {
		return ref, nil
	}

	if siteID != "" {
		record, err := p.sites.Get(ctx, siteID)
		if err != nil {
			return llm.ModelRef{}, err
		}
		if ref, ok := pick(record.Defaults.ModelProfiles, role); ok {
			return ref, nil
		}
	}

	global, err := p.store.List(ctx)
	if err != nil {
		return llm.ModelRef{}, err
	}
	if ref, ok := pick(global, role); ok {
		return ref, nil
	}

	return p.defaults.Default(role)
}

func (p *Profiles) Global(ctx context.Context) (map[llm.Role]llm.ModelRef, error) {
	return p.store.List(ctx)
}

func (p *Profiles) Set(ctx context.Context, role llm.Role, ref llm.ModelRef) error {
	if !role.Valid() {
		return errors.New(errors.Invalid, "a model profile role must be one of the known roles").WithDetail("role", string(role))
	}
	if !ref.Valid() {
		return errors.New(errors.Invalid, "a model profile must name a provider and a model").WithDetail("role", string(role))
	}
	return p.store.Set(ctx, role, ref, p.clock.Now().UTC().Truncate(time.Second))
}

func (p *Profiles) Clear(ctx context.Context, role llm.Role) error {
	return p.store.Delete(ctx, role)
}

func pick(profiles map[llm.Role]llm.ModelRef, role llm.Role) (llm.ModelRef, bool) {
	ref, ok := profiles[role]
	if !ok || !ref.Valid() {
		return llm.ModelRef{}, false
	}
	return ref, true
}
