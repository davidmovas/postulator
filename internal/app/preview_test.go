package app

import (
	"context"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type oneClient struct {
	client *wp.Client
	siteID string
}

func (o oneClient) Client(_ context.Context, siteID string) (*wp.Client, error) {
	if siteID != o.siteID {
		return nil, errors.New(errors.NotFound, "site not found").WithDetail("siteId", siteID)
	}
	return o.client, nil
}

func clientOver(t *testing.T, server *wptest.Server) *wp.Client {
	t.Helper()

	client, err := wp.New(wp.Config{
		BaseURL:       server.URL(),
		Username:      wptest.DefaultUser,
		AppPassword:   wptest.DefaultPassword,
		AllowInsecure: true,
	}, wp.WithRateLimit(0), wp.WithBackoff(func(int) time.Duration { return 0 }))
	if err != nil {
		t.Fatalf("wp.New: %v", err)
	}
	return client
}

func TestPreviewIssuerAsksTheSitesPlugin(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	draft := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein", Status: "draft"})[0]
	issuer := previewIssuer{clients: oneClient{client: clientOver(t, server), siteID: "s1"}}

	issued, err := issuer.IssuePreview(t.Context(), "s1", draft.ID)
	if err != nil {
		t.Fatalf("IssuePreview: %v", err)
	}
	stored, _ := server.Lookup(draft.ID)
	if issued.URL == "" || !issued.ExpiresAt.Equal(stored.PreviewExpires) {
		t.Errorf("issued = %+v, stored expiry %s", issued, stored.PreviewExpires)
	}

	if _, err = issuer.IssuePreview(t.Context(), "s2", draft.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("IssuePreview on an unknown site = %v", err)
	}
}

func TestPreviewIssuerReportsAMissingPlugin(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithoutPlugin())
	draft := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein", Status: "draft"})[0]
	issuer := previewIssuer{clients: oneClient{client: clientOver(t, server), siteID: "s1"}}

	if _, err := issuer.IssuePreview(t.Context(), "s1", draft.ID); !wp.IsPluginMissing(err) {
		t.Fatalf("IssuePreview without the plugin = %v, want plugin_missing", err)
	}
}
