package sqlitetest

import (
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

var Stamp = time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC)

func Site(t testing.TB, store *sqlite.Store, name string) site.Site {
	t.Helper()

	record := site.Site{
		ID:        id.New(),
		Name:      name,
		BaseURL:   "https://" + name + ".example.com",
		Username:  "editor",
		Status:    site.StatusActive,
		Plugin:    site.PluginState{Capabilities: []string{}},
		Defaults:  site.Defaults{ModelProfiles: map[llm.Role]llm.ModelRef{}},
		CreatedAt: Stamp,
		UpdatedAt: Stamp,
	}
	record.SecretRef = site.SecretRef(record.ID)
	if err := sqlite.NewSiteRepo(store).Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the site fixture: %v", err)
	}
	return record
}
