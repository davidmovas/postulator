package sites_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/secrets"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/applicationtest"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type probeStub struct {
	seen    site.Candidate
	reached site.Reachability
	err     error
}

func (p *probeStub) TestConnection(_ context.Context, candidate site.Candidate) (site.Reachability, error) {
	p.seen = candidate
	if p.err != nil {
		return site.Reachability{}, p.err
	}
	return p.reached, nil
}

type harness struct {
	service *sites.Service
	store   *sqlite.Store
	secrets *secrets.Store
	clock   *clock.Fake
	events  *applicationtest.Recorder
	probe   *probeStub
}

func newHarness(t *testing.T) harness {
	t.Helper()

	store := sqlitetest.Open(t)
	clk := clock.NewFake(time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC))
	vault := secrets.NewStore(sqlite.NewSecretsRepo(store, clk), sqlitetest.Key())
	recorder := &applicationtest.Recorder{}
	reach := &probeStub{reached: site.Reachability{Reach: site.ReachOK, Message: "the site answered", HasPlugin: true}}
	return harness{
		service: sites.New(sqlite.NewSiteRepo(store), vault, store, reach, recorder, clk),
		store:   store,
		secrets: vault,
		clock:   clk,
		events:  recorder,
		probe:   reach,
	}
}

func TestTestConnectionProbesCandidateCredentialsBeforeThereIsARow(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	answered, err := h.service.TestConnection(t.Context(), sites.TestConnectionRequest{
		BaseURL: " https://Shop.example.com/ ", Username: " editor ", Password: "abcd efgh",
	})
	if err != nil {
		t.Fatalf("TestConnection: %v", err)
	}

	if h.probe.seen.BaseURL != "https://shop.example.com" || h.probe.seen.Username != "editor" {
		t.Fatalf("the probe received %v", h.probe.seen)
	}
	if h.probe.seen.Password != "abcd efgh" || h.probe.seen.SiteID != "" {
		t.Fatalf("the probe received the wrong candidate: %v", h.probe.seen)
	}
	if answered.Reachability.Reach != string(site.ReachOK) || !answered.Reachability.HasPlugin {
		t.Fatalf("TestConnection = %+v", answered)
	}

	encoded, marshalErr := json.Marshal(answered)
	if marshalErr != nil {
		t.Fatalf("Marshal: %v", marshalErr)
	}
	if strings.Contains(string(encoded), "abcd efgh") {
		t.Fatalf("the response carries the password: %s", encoded)
	}
	for _, key := range []string{`"reach":"ok"`, `"message"`, `"hasPlugin":true`, `"suggestedBaseUrl"`} {
		if !strings.Contains(string(encoded), key) {
			t.Errorf("view lacks %s: %s", key, encoded)
		}
	}
}

func TestTestConnectionProbesAStoredSiteAndTheEditsOverIt(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	created, err := h.service.Create(t.Context(), sites.CreateRequest{
		Name: "Shop", BaseURL: "https://shop.example.com", Username: "editor", Password: "abcd efgh",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if _, err = h.service.TestConnection(t.Context(), sites.TestConnectionRequest{SiteID: created.Site.ID}); err != nil {
		t.Fatalf("TestConnection: %v", err)
	}
	if h.probe.seen.SiteID != created.Site.ID || h.probe.seen.BaseURL != "https://shop.example.com" ||
		h.probe.seen.Username != "editor" || h.probe.seen.Password != "" {
		t.Fatalf("the probe received %v", h.probe.seen)
	}

	if _, err = h.service.TestConnection(t.Context(), sites.TestConnectionRequest{
		SiteID: created.Site.ID, BaseURL: "http://staging.example.com", Username: "other",
		Password: "next pass", AllowInsecure: ptr(true),
	}); err != nil {
		t.Fatalf("TestConnection with edits: %v", err)
	}
	if h.probe.seen.BaseURL != "http://staging.example.com" || h.probe.seen.Username != "other" ||
		h.probe.seen.Password != "next pass" || !h.probe.seen.AllowInsecure {
		t.Fatalf("the probe received %v", h.probe.seen)
	}
}

func TestTestConnectionRefusesWhatItCannotProbe(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		request sites.TestConnectionRequest
		want    errors.Code
	}{
		{name: "nothing at all", request: sites.TestConnectionRequest{}, want: errors.Invalid},
		{
			name:    "no base url",
			request: sites.TestConnectionRequest{Username: "editor", Password: "abcd efgh"},
			want:    errors.Invalid,
		},
		{
			name:    "no user name",
			request: sites.TestConnectionRequest{BaseURL: "https://shop.example.com", Password: "abcd efgh"},
			want:    errors.Invalid,
		},
		{
			name:    "no password",
			request: sites.TestConnectionRequest{BaseURL: "https://shop.example.com", Username: "editor"},
			want:    errors.Invalid,
		},
		{
			name: "an insecure base url that was not allowed",
			request: sites.TestConnectionRequest{
				BaseURL: "http://shop.example.com", Username: "editor", Password: "abcd efgh",
			},
			want: errors.Invalid,
		},
		{
			name:    "a site that does not exist",
			request: sites.TestConnectionRequest{SiteID: "6f3b2a11-0c9d-4e7a-8b25-1f4c6d7e8a90"},
			want:    errors.NotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			if _, err := h.service.TestConnection(t.Context(), tc.request); !errors.IsCode(err, tc.want) {
				t.Fatalf("TestConnection = %v, want %s", err, tc.want)
			}
		})
	}
}

func TestTestConnectionCarriesTheProbeFailure(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.probe.err = errors.New(errors.Locked, "the vault is locked")

	_, err := h.service.TestConnection(t.Context(), sites.TestConnectionRequest{
		BaseURL: "https://shop.example.com", Username: "editor", Password: "abcd efgh",
	})
	if !errors.IsCode(err, errors.Locked) {
		t.Fatalf("TestConnection = %v, want %s", err, errors.Locked)
	}
}

func TestEveryWriteAnnouncesTheSiteItChanged(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	created, err := h.service.Create(t.Context(), sites.CreateRequest{
		Name: "Shop", BaseURL: "https://shop.example.com", Username: "editor", Password: "abcd efgh",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	cases := []struct {
		name string
		call func() error
	}{
		{
			name: "create",
			call: func() error { return nil },
		},
		{
			name: "update",
			call: func() error {
				_, updateErr := h.service.Update(t.Context(), sites.UpdateRequest{ID: created.Site.ID, Name: ptr("Shop 2")})
				return updateErr
			},
		},
		{
			name: "delete",
			call: func() error {
				_, deleteErr := h.service.Delete(t.Context(), sites.DeleteRequest{ID: created.Site.ID})
				return deleteErr
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h.events.Reset()
			if err := tc.call(); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if tc.name == "create" {
				h.events.Reset()
				again, createErr := h.service.Create(t.Context(), sites.CreateRequest{
					Name: "Other", BaseURL: "https://other.example.com", Username: "editor",
				})
				if createErr != nil {
					t.Fatalf("Create: %v", createErr)
				}
				assertSiteChanged(t, h.events.Events(), again.Site.ID)
				return
			}
			assertSiteChanged(t, h.events.Events(), created.Site.ID)
		})
	}
}

func assertSiteChanged(t *testing.T, recorded []applicationtest.Event, siteID string) {
	t.Helper()

	if len(recorded) != 1 {
		t.Fatalf("recorded %+v, want one event", recorded)
	}
	if recorded[0].Type != events.SitesChanged {
		t.Fatalf("type = %q, want %q", recorded[0].Type, events.SitesChanged)
	}
	payload, ok := recorded[0].Payload.(events.SitesChangedPayload)
	if !ok {
		t.Fatalf("payload = %T, want events.SitesChangedPayload", recorded[0].Payload)
	}
	if payload.SiteID != siteID {
		t.Fatalf("siteId = %q, want %q", payload.SiteID, siteID)
	}
}

func ptr[T any](v T) *T {
	return &v
}

func TestCreateStoresTheSiteAndItsPassword(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	created, err := h.service.Create(t.Context(), sites.CreateRequest{Name: " Shop ", BaseURL: "https://Shop.example.com/", Username: "editor", Password: "abcd efgh"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got := created.Site
	if got.ID == "" || got.Name != "Shop" || got.BaseURL != "https://shop.example.com" || got.Username != "editor" || got.Status != "active" || got.AllowInsecure {
		t.Errorf("Create = %+v", got)
	}
	if got.CreatedAt.String() != "2026-09-18T09:00:00Z" || got.UpdatedAt.String() != got.CreatedAt.String() {
		t.Errorf("timestamps = %s / %s", got.CreatedAt, got.UpdatedAt)
	}

	password, err := h.secrets.Get(t.Context(), site.SecretRef(got.ID))
	if err != nil || password != "abcd efgh" {
		t.Errorf("stored password = %q, %v", password, err)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	for _, key := range []string{`"id"`, `"baseUrl"`, `"allowInsecure"`, `"plugin":{"installed":false`, `"capabilities":[]`, `"defaults":{"templateId":null`, `"modelProfiles":{}`, `"createdAt":"2026-09-18T09:00:00Z"`} {
		if !strings.Contains(string(encoded), key) {
			t.Errorf("view lacks %s: %s", key, encoded)
		}
	}
	if strings.Contains(string(encoded), "secret") || strings.Contains(string(encoded), "abcd") {
		t.Errorf("the view must not carry the secret: %s", encoded)
	}
}

func TestCreateWithoutAPasswordStoresNoSecret(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	created, err := h.service.Create(t.Context(), sites.CreateRequest{Name: "Shop", BaseURL: "http://shop.local", AllowInsecure: true})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err = h.secrets.Get(t.Context(), site.SecretRef(created.Site.ID)); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("secret code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestCreateRejectsBadInput(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	cases := []struct {
		name string
		req  sites.CreateRequest
	}{
		{name: "blank name", req: sites.CreateRequest{Name: " ", BaseURL: "https://a.example.com"}},
		{name: "http without allow", req: sites.CreateRequest{Name: "A", BaseURL: "http://a.example.com"}},
		{name: "no url", req: sites.CreateRequest{Name: "A"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := h.service.Create(t.Context(), tc.req)
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("code = %q, want INVALID", errors.CodeOf(err))
			}
		})
	}
}

func TestUpdateAppliesOnlyTheGivenFields(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	created, err := h.service.Create(t.Context(), sites.CreateRequest{Name: "Shop", BaseURL: "https://shop.example.com", Password: "one"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	h.clock.Advance(time.Hour)

	updated, err := h.service.Update(t.Context(), sites.UpdateRequest{
		ID: created.Site.ID, Name: ptr("Shop Two"), Status: ptr("paused"), Password: ptr("two"),
		Defaults: &sites.Defaults{TemplateID: nil, ModelProfiles: map[string]llm.ModelRef{"writer": {Provider: "openai", Model: "gpt"}}},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	got := updated.Site
	if got.Name != "Shop Two" || got.Status != "paused" || got.BaseURL != "https://shop.example.com" || got.Defaults.ModelProfiles["writer"].Model != "gpt" {
		t.Errorf("Update = %+v", got)
	}
	if got.UpdatedAt.String() != "2026-09-18T10:00:00Z" || got.CreatedAt.String() != "2026-09-18T09:00:00Z" {
		t.Errorf("timestamps = %s / %s", got.CreatedAt, got.UpdatedAt)
	}
	if password, getErr := h.secrets.Get(t.Context(), site.SecretRef(got.ID)); getErr != nil || password != "two" {
		t.Errorf("rotated password = %q, %v", password, getErr)
	}

	if _, err = h.service.Update(t.Context(), sites.UpdateRequest{ID: created.Site.ID, Password: ptr("")}); err != nil {
		t.Fatalf("Update clearing the password: %v", err)
	}
	if _, err = h.secrets.Get(t.Context(), site.SecretRef(got.ID)); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("an empty password must delete the secret, got %v", err)
	}

	if _, err = h.service.Update(t.Context(), sites.UpdateRequest{ID: created.Site.ID, Status: ptr("sleeping")}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("bad status code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.Update(t.Context(), sites.UpdateRequest{ID: created.Site.ID, BaseURL: ptr("http://shop.example.com")}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("http without allow code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.Update(t.Context(), sites.UpdateRequest{ID: "missing", Name: ptr("x")}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("missing site code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = h.service.Update(t.Context(), sites.UpdateRequest{ID: created.Site.ID, Defaults: &sites.Defaults{TemplateID: ptr("no-such-template")}}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("unknown default template code = %q, want INVALID", errors.CodeOf(err))
	}
}

func TestDeleteCascadesAndRemovesTheSecret(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	created, err := h.service.Create(t.Context(), sites.CreateRequest{Name: "Shop", BaseURL: "https://shop.example.com", Password: "one"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	entity := sqlitetest.Entity(t, h.store, created.Site.ID, "Shoes")
	page := sqlitetest.Page(t, h.store, created.Site.ID, "/shoes/")

	if _, err = h.service.Delete(t.Context(), sites.DeleteRequest{ID: created.Site.ID}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err = h.service.Get(t.Context(), sites.GetRequest{ID: created.Site.ID}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Get after delete code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if _, err = sqlite.NewEntityRepo(h.store).Get(t.Context(), entity.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("entity must cascade, got %v", err)
	}
	if _, err = sqlite.NewPageRepo(h.store).Get(t.Context(), page.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("page must cascade, got %v", err)
	}
	if _, err = h.secrets.Get(t.Context(), site.SecretRef(created.Site.ID)); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("secret must be removed, got %v", err)
	}
	if _, err = h.service.Delete(t.Context(), sites.DeleteRequest{ID: created.Site.ID}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Delete twice code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestGetAndList(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	for i, name := range []string{"alpha", "bravo", "charlie"} {
		created, err := h.service.Create(t.Context(), sites.CreateRequest{Name: name, BaseURL: "https://" + name + ".example.com"})
		if err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
		if i == 1 {
			if _, err = h.service.Update(t.Context(), sites.UpdateRequest{ID: created.Site.ID, Status: ptr("paused")}); err != nil {
				t.Fatalf("Update: %v", err)
			}
		}
		h.clock.Advance(time.Minute)
	}

	page, err := h.service.List(t.Context(), sites.ListRequest{ListRequest: dto.ListRequest{Limit: 2}})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Items) != 2 || page.Items[0].Name != "alpha" || !page.HasMore || page.Next == "" {
		t.Fatalf("List = %+v", page)
	}
	rest, err := h.service.List(t.Context(), sites.ListRequest{ListRequest: dto.ListRequest{Cursor: string(page.Next), Limit: 2}})
	if err != nil || len(rest.Items) != 1 || rest.Items[0].Name != "charlie" {
		t.Errorf("List after = %+v, %v", rest, err)
	}
	paused, err := h.service.List(t.Context(), sites.ListRequest{Status: "paused"})
	if err != nil || len(paused.Items) != 1 || paused.Items[0].Name != "bravo" {
		t.Errorf("List paused = %+v, %v", paused, err)
	}
	byName, err := h.service.List(t.Context(), sites.ListRequest{ListRequest: dto.ListRequest{Sort: &dto.Sort{Field: "name", Desc: true}}})
	if err != nil || len(byName.Items) != 3 || byName.Items[0].Name != "charlie" {
		t.Errorf("List by name desc = %+v, %v", byName, err)
	}
	if _, err = h.service.List(t.Context(), sites.ListRequest{ListRequest: dto.ListRequest{Sort: &dto.Sort{Field: "age"}}}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("unknown sort code = %q, want INVALID", errors.CodeOf(err))
	}
	if _, err = h.service.List(t.Context(), sites.ListRequest{Status: "sleeping"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("unknown status code = %q, want INVALID", errors.CodeOf(err))
	}

	got, err := h.service.Get(t.Context(), sites.GetRequest{ID: page.Items[0].ID})
	if err != nil || got.Site.Name != "alpha" {
		t.Errorf("Get = %+v, %v", got, err)
	}
}
