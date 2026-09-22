package steps

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

const (
	NameSyncSite = string(run.StepSyncSite)

	CapabilityBulk = "bulk"

	SourcePlugin = "plugin"
	SourceCore   = "core"

	checkpointSync = "sync"

	syncStepTimeout = 10 * time.Minute
)

var coreTypes = []wp.ItemType{wp.TypePage, wp.TypePost}

type SiteSyncResult struct {
	StartedAt time.Time `json:"startedAt"`
	Source    string    `json:"source"`
	Cursor    string    `json:"cursor"`
	Batches   int       `json:"batches"`
	Pulled    int       `json:"pulled"`
	Created   int       `json:"created"`
	Updated   int       `json:"updated"`
	Drifted   int       `json:"drifted"`
	Archived  int       `json:"archived"`
	Done      bool      `json:"done"`
}

type coreCursor struct {
	Type string `json:"type"`
	Page int    `json:"page"`
}

type pulledItem struct {
	Modified    time.Time
	Type        pagemap.WPType
	Path        string
	Slug        string
	Status      string
	Title       string
	H1          string
	ContentHash string
	Meta        wp.ContentMeta
	Links       []wp.ContentLink
	WPID        int64
	ParentWPID  int64
}

func SyncSite(deps Deps) run.StepDef {
	return run.StepDef{
		Name:    NameSyncSite,
		Retry:   run.RetryPolicy{Max: 3},
		Timeout: syncStepTimeout,
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			state, _, err := run.Get[SiteSyncResult](sc.Check, checkpointSync)
			if err != nil {
				return run.Result{}, err
			}
			owner, err := deps.Sites.Get(ctx, sc.Run.SiteID)
			if err != nil {
				return run.Result{}, err
			}
			client, err := clientFor(ctx, deps, sc.Run.SiteID)
			if err != nil {
				return run.Result{}, err
			}

			bulk, err := adoptManifest(ctx, deps, client, owner)
			if err != nil {
				return run.Result{}, err
			}

			source := SourceCore
			if bulk {
				source = SourcePlugin
			}
			if state.Source != "" && state.Source != source {
				state = SiteSyncResult{}
			}
			state.Source = source
			if state.StartedAt.IsZero() {
				state.StartedAt = deps.now()
			}

			batch, next, err := pull(ctx, client, state, batchSize(deps), hostOf(owner.BaseURL))
			if err != nil {
				return run.Result{}, err
			}

			if reconcileErr := reconcile(ctx, deps, owner, batch, &state); reconcileErr != nil {
				return run.Result{}, reconcileErr
			}

			state.Batches++
			state.Pulled += len(batch)
			state.Cursor = next
			state.Done = next == ""

			if state.Done {
				if archiveErr := archiveAbsent(ctx, deps, owner.ID, &state); archiveErr != nil {
					return run.Result{}, archiveErr
				}
				if parentErr := linkParents(ctx, deps, owner.ID); parentErr != nil {
					return run.Result{}, parentErr
				}
				if resolveErr := resolveLinks(ctx, deps, owner.ID); resolveErr != nil {
					return run.Result{}, resolveErr
				}
				if announceErr := announcePages(deps, owner.ID); announceErr != nil {
					return run.Result{}, announceErr
				}
			}

			checkpoint := run.NewCheckpoint()
			if setErr := run.Set(checkpoint, checkpointSync, state); setErr != nil {
				return run.Result{}, setErr
			}

			blob, err := encode(state, "site sync result")
			if err != nil {
				return run.Result{}, err
			}

			result := run.Result{
				Artifacts:  []run.Artifact{{Kind: run.ArtifactSyncResult, Blob: blob}},
				Checkpoint: checkpoint,
				Message: "pulled " + strconv.Itoa(state.Pulled) + " items from " + owner.Name +
					" through the " + state.Source,
			}
			if !state.Done {
				result.Next = run.TransitionWait
				result.WakeAt = deps.now()
			}
			return result, nil
		},
	}
}

func batchSize(deps Deps) int {
	if deps.BatchSize <= 0 {
		return DefaultBatchSize
	}
	return deps.BatchSize
}

func adoptManifest(ctx context.Context, deps Deps, client *wp.Client, owner site.Site) (bool, error) {
	capabilities, err := client.Capabilities(ctx)
	if err != nil {
		if !wp.IsPluginMissing(err) {
			return false, err
		}
		return false, adoptPlugin(ctx, deps, owner, site.PluginState{Capabilities: []string{}})
	}

	return capabilities.Has(CapabilityBulk), adoptPlugin(ctx, deps, owner, site.PluginState{
		Installed:    true,
		Version:      capabilities.Version,
		Capabilities: capabilities.Names,
		SEOPlugin:    capabilities.SEOPlugin,
	})
}

func adoptPlugin(ctx context.Context, deps Deps, owner site.Site, state site.PluginState) error {
	if deps.SiteWriter == nil || samePlugin(owner.Plugin, state) {
		return nil
	}

	next := owner
	next.Plugin = state
	next.UpdatedAt = deps.now()
	return deps.SiteWriter.Update(ctx, next)
}

func samePlugin(current, next site.PluginState) bool {
	if current.Installed != next.Installed || current.Version != next.Version ||
		current.SEOPlugin != next.SEOPlugin || len(current.Capabilities) != len(next.Capabilities) {
		return false
	}
	for i := range current.Capabilities {
		if current.Capabilities[i] != next.Capabilities[i] {
			return false
		}
	}
	return true
}

func pull(ctx context.Context, client *wp.Client, state SiteSyncResult, limit int, host string) ([]pulledItem, string, error) {
	if state.Source == SourcePlugin {
		return pullBulk(ctx, client, state.Cursor, limit)
	}
	return pullCore(ctx, client, state.Cursor, limit, host)
}

func pullBulk(ctx context.Context, client *wp.Client, cursor string, limit int) ([]pulledItem, string, error) {
	page, err := client.ListContent(ctx, wp.ContentQuery{Cursor: cursor, Limit: limit})
	if err != nil {
		return nil, "", err
	}

	out := make([]pulledItem, 0, len(page.Items))
	for i := range page.Items {
		item := &page.Items[i]
		path, normalizeErr := pagemap.NormalizePath(item.Path)
		if normalizeErr != nil {
			continue
		}
		out = append(out, pulledItem{
			WPID: item.ID, Type: pagemap.WPType(item.Type), Path: path, Slug: item.Slug,
			Status: item.Status, Title: item.Title, H1: item.H1, ContentHash: item.ContentHash,
			Modified: item.Modified, Meta: item.Meta, Links: item.Links,
		})
	}

	next := ""
	if page.NextCursor != nil {
		next = *page.NextCursor
	}
	return out, next, nil
}

func pullCore(ctx context.Context, client *wp.Client, cursor string, limit int, host string) ([]pulledItem, string, error) {
	position, err := decodeCore(cursor)
	if err != nil {
		return nil, "", err
	}

	for index := typeIndex(position.Type); index < len(coreTypes); index++ {
		itemType := coreTypes[index]
		page, listErr := client.ListItems(ctx, itemType, wp.ListQuery{
			Page: position.Page, PerPage: limit, Status: editableStatuses,
		})
		if listErr != nil {
			return nil, "", listErr
		}
		if len(page.Items) == 0 {
			position.Page = 1
			continue
		}

		out := make([]pulledItem, 0, len(page.Items))
		for i := range page.Items {
			converted, ok := fromCore(page.Items[i], itemType, host)
			if ok {
				out = append(out, converted)
			}
		}

		if page.HasMore {
			return out, encodeCore(coreCursor{Type: string(itemType), Page: position.Page + 1}), nil
		}
		if index+1 < len(coreTypes) {
			return out, encodeCore(coreCursor{Type: string(coreTypes[index+1]), Page: 1}), nil
		}
		return out, "", nil
	}
	return nil, "", nil
}

func typeIndex(name string) int {
	for i := range coreTypes {
		if string(coreTypes[i]) == name {
			return i
		}
	}
	return 0
}

func decodeCore(cursor string) (coreCursor, error) {
	if cursor == "" {
		return coreCursor{Type: string(coreTypes[0]), Page: 1}, nil
	}

	var decoded coreCursor
	if err := json.Unmarshal([]byte(cursor), &decoded); err != nil {
		return coreCursor{}, errors.Wrap(err, errors.Invalid, "the stored sync cursor is not readable")
	}
	if decoded.Page < 1 {
		decoded.Page = 1
	}
	return decoded, nil
}

func encodeCore(position coreCursor) string {
	encoded, err := json.Marshal(position)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func fromCore(item wp.Item, itemType wp.ItemType, host string) (pulledItem, bool) {
	path, internal := pagemap.InternalPath(item.Link, host)
	if !internal {
		return pulledItem{}, false
	}

	if queryPermalink(item.Link) {
		path = ""
	}

	pulled := pulledItem{
		WPID: item.ID, ParentWPID: item.Parent, Type: pagemap.WPType(itemType), Path: path,
		Slug: item.Slug, Status: item.Status, Title: item.Title,
		ContentHash: wp.ContentHash(item.Content), Modified: item.Modified,
	}
	if doc, err := content.Parse(item.Content); err == nil {
		pulled.H1 = headingOne(doc)
		pulled.Links = internalLinks(doc, host)
	}
	return pulled, true
}

func queryPermalink(link string) bool {
	parsed, err := url.Parse(strings.TrimSpace(link))
	if err != nil {
		return false
	}
	return parsed.RawQuery != ""
}

func resolveDraftPaths(batch []pulledItem, byWPID map[int64]pagemap.Page) []pulledItem {
	known := make(map[int64]string, len(byWPID)+len(batch))
	for wpID := range byWPID {
		known[wpID] = byWPID[wpID].Path
	}
	for i := range batch {
		if batch[i].Path != "" {
			known[batch[i].WPID] = batch[i].Path
		}
	}

	out := make([]pulledItem, 0, len(batch))
	for i := range batch {
		if batch[i].Path == "" {
			path, ok := draftPath(batch[i], known)
			if !ok {
				continue
			}
			batch[i].Path = path
		}
		out = append(out, batch[i])
	}
	return out
}

func draftPath(item pulledItem, known map[int64]string) (string, bool) {
	if item.Slug == "" {
		return "", false
	}

	base := "/"
	if item.ParentWPID != 0 {
		parent, ok := known[item.ParentWPID]
		if !ok {
			return "", false
		}
		base = parent
	}

	path, err := pagemap.NormalizePath(base + item.Slug + "/")
	if err != nil {
		return "", false
	}
	return path, true
}

func headingOne(doc *content.Document) string {
	for _, heading := range doc.Headings() {
		if heading.Data == "h1" {
			return content.TextOf(heading)
		}
	}
	return ""
}

func internalLinks(doc *content.Document, host string) []wp.ContentLink {
	found := doc.Links()

	out := make([]wp.ContentLink, 0, len(found))
	for i := range found {
		path, internal := pagemap.InternalPath(found[i].Href, host)
		if !internal {
			continue
		}
		out = append(out, wp.ContentLink{Href: path, Anchor: found[i].Anchor})
	}
	return out
}

func reconcile(ctx context.Context, deps Deps, owner site.Site, batch []pulledItem, state *SiteSyncResult) error {
	if len(batch) == 0 {
		return nil
	}

	pages, err := deps.Pages.ListBySite(ctx, owner.ID)
	if err != nil {
		return err
	}

	byPath := make(map[string]pagemap.Page, len(pages))
	byWPID := make(map[int64]pagemap.Page, len(pages))
	for i := range pages {
		byPath[pages[i].Path] = pages[i]
		if pages[i].WPID != nil {
			byWPID[*pages[i].WPID] = pages[i]
		}
	}

	batch = resolveDraftPaths(batch, byWPID)
	if len(batch) == 0 {
		return nil
	}

	now := deps.now()
	apply := func(c context.Context) error {
		for i := range batch {
			current, known := match(batch[i], byWPID, byPath)
			next, drifted := merge(current, known, batch[i], owner.ID, now)

			if known {
				if updateErr := deps.Pages.Update(c, next); updateErr != nil {
					return updateErr
				}
				state.Updated++
			} else {
				if insertErr := deps.Pages.Insert(c, next); insertErr != nil {
					return insertErr
				}
				state.Created++
			}
			if drifted {
				state.Drifted++
			}
			byPath[next.Path] = next
			byWPID[*next.WPID] = next

			linkErr := deps.Links.ReplaceForPage(c, next.ID, pulledLinks(next, byPath, batch[i].Links, now))
			if linkErr != nil {
				return linkErr
			}
		}
		return nil
	}

	if deps.UnitOfWork == nil {
		return apply(ctx)
	}
	return deps.UnitOfWork.Do(ctx, apply)
}

func match(item pulledItem, byWPID map[int64]pagemap.Page, byPath map[string]pagemap.Page) (pagemap.Page, bool) {
	if page, ok := byWPID[item.WPID]; ok {
		return page, true
	}
	if page, ok := byPath[item.Path]; ok {
		return page, true
	}
	return pagemap.Page{}, false
}

func merge(current pagemap.Page, known bool, item pulledItem, siteID string,
	now time.Time) (next pagemap.Page, drifted bool) {
	next = current
	if !known {
		next = pagemap.Page{ID: id.New(), SiteID: siteID, CreatedAt: now}
	}

	drifted = known && next.ContentHash != "" && next.ContentHash != item.ContentHash

	if !authored(next) {
		next.Path = item.Path
		next.Slug = pagemap.Slug(item.Path)
		next.Title = item.Title
		next.H1 = item.H1
		next.MetaTitle = item.Meta.Title
		next.MetaDescription = item.Meta.Description
		next.Canonical = item.Meta.Canonical
		next.Status = statusFor(item.Status)
	}
	next.WPType = wpTypeOrPage(item.Type)
	next.WPID = &item.WPID
	next.Observed = pagemap.Observed{
		Link: item.Path, Slug: item.Slug, Status: item.Status, Title: item.Title, H1: item.H1,
	}
	next.Drift = drifted
	next.LastSyncedAt = &now
	next.UpdatedAt = now
	if !item.Modified.IsZero() {
		modified := item.Modified.UTC()
		next.WPModifiedAt = &modified
	}
	return next, drifted
}

func authored(page pagemap.Page) bool {
	return page.ContentHash != ""
}

func wpTypeOrPage(itemType pagemap.WPType) pagemap.WPType {
	if itemType.Valid() {
		return itemType
	}
	return pagemap.WPPage
}

func statusFor(wpStatus string) pagemap.Status {
	if wpStatus == "publish" {
		return pagemap.StatusPublished
	}
	return pagemap.StatusExists
}

func pulledLinks(page pagemap.Page, byPath map[string]pagemap.Page, links []wp.ContentLink, at time.Time) []pagemap.PageLink {
	out := make([]pagemap.PageLink, 0, len(links))
	for i := range links {
		path, err := pagemap.NormalizePath(links[i].Href)
		if err != nil {
			continue
		}

		link := pagemap.PageLink{
			ID: id.New(), SiteID: page.SiteID, FromPageID: page.ID, ToURL: path,
			AnchorText: links[i].Anchor, Origin: pagemap.OriginObserved, ObservedAt: at,
		}
		if target, ok := byPath[path]; ok {
			link.ToPageID = &target.ID
		}
		out = append(out, link)
	}
	return out
}

func resolveLinks(ctx context.Context, deps Deps, siteID string) error {
	pages, err := deps.Pages.ListBySite(ctx, siteID)
	if err != nil {
		return err
	}

	index := pagemap.NewIndex(pages)
	pending := make(map[string][]pagemap.PageLink)
	for i := range pages {
		links, listErr := deps.Links.ListForPage(ctx, pages[i].ID)
		if listErr != nil {
			return listErr
		}
		if resolveInto(links, index) {
			pending[pages[i].ID] = links
		}
	}
	if len(pending) == 0 {
		return nil
	}

	apply := func(c context.Context) error {
		for pageID, links := range pending {
			if replaceErr := deps.Links.ReplaceForPage(c, pageID, links); replaceErr != nil {
				return replaceErr
			}
		}
		return nil
	}
	if deps.UnitOfWork == nil {
		return apply(ctx)
	}
	return deps.UnitOfWork.Do(ctx, apply)
}

func resolveInto(links []pagemap.PageLink, index pagemap.Index) bool {
	changed := false
	for i := range links {
		if links[i].ToPageID != nil {
			continue
		}
		target, ok := index.ByPath(links[i].ToURL)
		if !ok || target.ID == links[i].FromPageID {
			continue
		}
		links[i].ToPageID = &target.ID
		changed = true
	}
	return changed
}

func archiveAbsent(ctx context.Context, deps Deps, siteID string, state *SiteSyncResult) error {
	pages, err := deps.Pages.ListBySite(ctx, siteID)
	if err != nil {
		return err
	}

	now := deps.now()
	stale := make([]pagemap.Page, 0)
	for i := range pages {
		page := pages[i]
		if page.WPID == nil || page.Status == pagemap.StatusArchived || !covered(state.Source, page.WPType) {
			continue
		}
		if page.LastSyncedAt != nil && !page.LastSyncedAt.Before(state.StartedAt) {
			continue
		}
		page.Status = pagemap.StatusArchived
		page.UpdatedAt = now
		stale = append(stale, page)
	}
	if len(stale) == 0 {
		return nil
	}

	apply := func(c context.Context) error {
		for i := range stale {
			if updateErr := deps.Pages.Update(c, stale[i]); updateErr != nil {
				return updateErr
			}
		}
		return nil
	}
	if deps.UnitOfWork != nil {
		err = deps.UnitOfWork.Do(ctx, apply)
	} else {
		err = apply(ctx)
	}
	if err != nil {
		return err
	}

	state.Archived += len(stale)
	return nil
}

func linkParents(ctx context.Context, deps Deps, siteID string) error {
	pages, err := deps.Pages.ListBySite(ctx, siteID)
	if err != nil {
		return err
	}

	index := pagemap.NewIndex(pages)
	now := deps.now()
	moved := make([]pagemap.Page, 0)
	for i := range pages {
		page := pages[i]
		var wanted *string
		if parent, ok := index.ByPath(pagemap.ParentPath(page.Path)); ok && parent.ID != page.ID {
			wanted = &parent.ID
		}
		if sameRef(page.ParentPageID, wanted) {
			continue
		}
		page.ParentPageID = wanted
		page.UpdatedAt = now
		moved = append(moved, page)
	}
	if len(moved) == 0 {
		return nil
	}

	apply := func(c context.Context) error {
		for i := range moved {
			if updateErr := deps.Pages.Update(c, moved[i]); updateErr != nil {
				return updateErr
			}
		}
		return nil
	}
	if deps.UnitOfWork == nil {
		return apply(ctx)
	}
	return deps.UnitOfWork.Do(ctx, apply)
}

func sameRef(current, wanted *string) bool {
	if current == nil || wanted == nil {
		return current == nil && wanted == nil
	}
	return *current == *wanted
}

func covered(source string, wpType pagemap.WPType) bool {
	if source == SourcePlugin {
		return true
	}
	for i := range coreTypes {
		if string(coreTypes[i]) == string(wpType) {
			return true
		}
	}
	return false
}

func announcePages(deps Deps, siteID string) error {
	if deps.Publisher == nil {
		return nil
	}
	return deps.Publisher.Publish(events.PagesChanged, events.PagesChangedPayload{SiteID: siteID})
}
