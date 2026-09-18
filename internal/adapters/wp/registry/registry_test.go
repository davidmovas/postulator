package registry_test

import (
	"context"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/registry"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const siteID = "5c31b2fe-7b6b-4a07-93d7-0d0b3c5a5f01"

type sites struct {
	record site.Site
	err    error
}

func (s *sites) Get(context.Context, string) (site.Site, error) {
	if s.err != nil {
		return site.Site{}, s.err
	}
	return s.record, nil
}

type vault struct {
	password string
	err      error
}

func (v vault) Get(context.Context, string) (string, error) {
	if v.err != nil {
		return "", v.err
	}
	return v.password, nil
}

func newSites(server *wptest.Server) *sites {
	record := site.Site{
		ID: siteID, Name: "Shop", BaseURL: server.URL(), Username: wptest.DefaultUser,
		Status: site.StatusActive, AllowInsecure: true,
	}
	record.SecretRef = site.SecretRef(record.ID)
	return &sites{record: record}
}

func TestClientIsBuiltOnceAndRebuiltOnChange(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	store := newSites(server)
	held := registry.New(store, vault{password: wptest.DefaultPassword}, wp.WithRateLimit(0))

	first, err := held.Client(t.Context(), siteID)
	if err != nil {
		t.Fatalf("Client: %v", err)
	}
	second, err := held.Client(t.Context(), siteID)
	if err != nil {
		t.Fatalf("Client again: %v", err)
	}
	if first != second {
		t.Fatal("the registry rebuilt a client whose site did not change")
	}

	store.record.UpdatedAt = store.record.UpdatedAt.Add(1)
	third, err := held.Client(t.Context(), siteID)
	if err != nil {
		t.Fatalf("Client after a change: %v", err)
	}
	if third == first {
		t.Fatal("the registry kept a client whose site changed")
	}
}

func TestClientReportsWhatItCannotDo(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)

	cases := []struct {
		name    string
		sites   func() *sites
		secrets vault
		want    errors.Code
	}{
		{
			name:  "the site is gone",
			sites: func() *sites { s := newSites(server); s.err = errors.New(errors.NotFound, "gone"); return s },
			want:  errors.NotFound,
		},
		{
			name:    "no password is stored",
			sites:   func() *sites { return newSites(server) },
			secrets: vault{err: errors.New(errors.NotFound, "no secret")},
			want:    errors.Unauthorized,
		},
		{
			name:    "the secret store is locked",
			sites:   func() *sites { return newSites(server) },
			secrets: vault{err: errors.New(errors.Locked, "locked")},
			want:    errors.Locked,
		},
		{
			name: "the base url is refused",
			sites: func() *sites {
				s := newSites(server)
				s.record.AllowInsecure = false
				return s
			},
			secrets: vault{password: wptest.DefaultPassword},
			want:    errors.Invalid,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			held := registry.New(tc.sites(), tc.secrets)
			if _, err := held.Client(t.Context(), siteID); !errors.IsCode(err, tc.want) {
				t.Fatalf("code = %q, want %q (err %v)", errors.CodeOf(err), tc.want, err)
			}
		})
	}
}

func TestProbeReadsTheManifest(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		opts      []wptest.Option
		installed bool
		seo       string
	}{
		{name: "the plugin answers", installed: true, seo: "yoast"},
		{name: "the plugin is absent", opts: []wptest.Option{wptest.WithoutPlugin()}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t, tc.opts...)
			store := newSites(server)
			held := registry.New(store, vault{password: wptest.DefaultPassword}, wp.WithRateLimit(0))

			state, err := held.Probe(t.Context(), store.record)
			if err != nil {
				t.Fatalf("Probe: %v", err)
			}
			if state.Installed != tc.installed || state.SEOPlugin != tc.seo {
				t.Fatalf("state = %+v", state)
			}
			if !tc.installed && len(state.Capabilities) != 0 {
				t.Fatalf("state = %+v", state)
			}
			if tc.installed && len(state.Capabilities) == 0 {
				t.Fatalf("state = %+v", state)
			}
		})
	}
}

func TestProbeCarriesTheClientFailure(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	store := newSites(server)
	held := registry.New(store, vault{err: errors.New(errors.Locked, "locked")})

	if _, err := held.Probe(t.Context(), store.record); !errors.IsCode(err, errors.Locked) {
		t.Fatalf("code = %q, want %q (err %v)", errors.CodeOf(err), errors.Locked, err)
	}
}
