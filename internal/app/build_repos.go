package app

import (
	"context"

	"github.com/davidmovas/postulator/internal/adapters/secrets"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/settings"
)

type repos struct {
	store         *sqlite.Store
	now           clock.Clock
	declarations  *settings.Registry
	settingsStore *sqlite.SettingsRepo
	values        *settings.Values
	unknown       []string
	secrets       *secrets.Store
	sites         *sqlite.SiteRepo
	entities      *sqlite.EntityRepo
	edges         *sqlite.EdgeRepo
	pages         *sqlite.PageRepo
	links         *sqlite.PageLinkRepo
	templates     *sqlite.TemplateRepo
	policies      *sqlite.LinkPolicyRepo
	overrides     *sqlite.ModelCatalogRepo
	profiles      *sqlite.ModelProfileRepo
	llmCalls      *sqlite.LLMCallRepo
	runs          *sqlite.RunRepo
	items         *sqlite.RunItemRepo
	artifacts     *sqlite.ArtifactRepo
	execs         *sqlite.StepExecRepo
	runEvents     *sqlite.RunEventRepo
	mappings      *sqlite.ImportMappingRepo
	schedules     *sqlite.ScheduleRepo
	actions       *sqlite.PendingActionRepo
	conversations *sqlite.ConversationRepo
	messages      *sqlite.MessageRepo
	toolCalls     *sqlite.ToolCallRepo
	history       *sqlite.ConversationHistoryRepo
	terms         *sqlite.TermRepo
}

func (c *Core) openStore(key []byte) (*sqlite.Store, error) {
	opened := sqlite.Config{Path: c.cfg.DatabasePath, Key: key, Recovery: c.cfg.recovery()}
	if c.cfg.Open != nil {
		return c.cfg.Open(opened)
	}
	return sqlite.Open(opened)
}

func openRepos(ctx context.Context, store *sqlite.Store, key []byte) (repos, error) {
	now := clock.System{}
	declarations := settings.Default()
	settingsStore := sqlite.NewSettingsRepo(store, now)
	values, unknown, err := LoadSettings(ctx, settingsStore, declarations)
	if err != nil {
		return repos{}, err
	}

	return repos{
		store:         store,
		now:           now,
		declarations:  declarations,
		settingsStore: settingsStore,
		values:        values,
		unknown:       unknown,
		secrets:       secrets.NewStore(sqlite.NewSecretsRepo(store, now), key),
		sites:         sqlite.NewSiteRepo(store),
		entities:      sqlite.NewEntityRepo(store),
		edges:         sqlite.NewEdgeRepo(store),
		pages:         sqlite.NewPageRepo(store),
		links:         sqlite.NewPageLinkRepo(store),
		templates:     sqlite.NewTemplateRepo(store),
		policies:      sqlite.NewLinkPolicyRepo(store),
		overrides:     sqlite.NewModelCatalogRepo(store),
		profiles:      sqlite.NewModelProfileRepo(store),
		llmCalls:      sqlite.NewLLMCallRepo(store),
		runs:          sqlite.NewRunRepo(store),
		items:         sqlite.NewRunItemRepo(store),
		artifacts:     sqlite.NewArtifactRepo(store),
		execs:         sqlite.NewStepExecRepo(store),
		runEvents:     sqlite.NewRunEventRepo(store),
		mappings:      sqlite.NewImportMappingRepo(store),
		schedules:     sqlite.NewScheduleRepo(store),
		actions:       sqlite.NewPendingActionRepo(store),
		conversations: sqlite.NewConversationRepo(store),
		messages:      sqlite.NewMessageRepo(store),
		toolCalls:     sqlite.NewToolCallRepo(store),
		history:       sqlite.NewConversationHistoryRepo(store),
		terms:         sqlite.NewTermRepo(store),
	}, nil
}
