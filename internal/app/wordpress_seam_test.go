package app_test

import (
	"path/filepath"
	"testing"

	"go.uber.org/zap/zaptest"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/sites"
	appsync "github.com/davidmovas/postulator/internal/application/sync"
)

func TestOpenBuildsEveryWordPressClientOverTheConfigsOptions(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	home := t.TempDir()
	core, err := app.Open(t.Context(), app.Config{
		DatabasePath: filepath.Join(home, "postulator.db"), KeyDir: home,
		WordPress: []wp.Option{wp.WithUserAgent("Harness/1"), wp.WithRateLimit(0)},
	}, zaptest.NewLogger(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := core.Close(); closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	})

	owner, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name: "Shop", BaseURL: server.URL(), Username: wptest.DefaultUser, Password: wptest.DefaultPassword, AllowInsecure: true,
	})
	if err != nil {
		t.Fatalf("create the site: %v", err)
	}
	server.ResetRequests()
	if _, err = core.Sync.CheckPlugin(t.Context(), appsync.CheckPluginRequest{SiteID: owner.Site.ID}); err != nil {
		t.Fatalf("check the plugin: %v", err)
	}

	requests := server.Requests()
	if len(requests) == 0 {
		t.Fatal("no request reached the site")
	}
	for _, recorded := range requests {
		if got := recorded.Header.Get("User-Agent"); got != "Harness/1" {
			t.Errorf("%s %s User-Agent = %q, want the config's %q", recorded.Method, recorded.Path, got, "Harness/1")
		}
	}
}
