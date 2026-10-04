package steps

import (
	"context"
	"maps"
	"slices"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

type siteKey struct {
	wpID int64
	term bool
}

func keyOf(wpType pagemap.WPType, wpID int64) siteKey {
	return siteKey{wpID: wpID, term: wpType.Term()}
}

type siteRows struct {
	byPath map[string]pagemap.Page
	byWPID map[siteKey]pagemap.Page
}

func rowsOf(pages []pagemap.Page) siteRows {
	rows := siteRows{
		byPath: make(map[string]pagemap.Page, len(pages)),
		byWPID: make(map[siteKey]pagemap.Page, len(pages)),
	}
	for i := range pages {
		rows.byPath[pages[i].Path] = pages[i]
		if pages[i].WPID != nil {
			rows.byWPID[keyOf(pages[i].WPType, *pages[i].WPID)] = pages[i]
		}
	}
	return rows
}

type reconciler struct {
	deps   Deps
	rows   siteRows
	state  *SiteSyncResult
	siteID string
	now    time.Time
}

func reconcile(ctx context.Context, deps Deps, owner site.Site, batch []pulledItem, state *SiteSyncResult) error {
	if len(batch) == 0 {
		return nil
	}

	pages, err := deps.Pages.ListBySite(ctx, owner.ID)
	if err != nil {
		return err
	}
	rows := rowsOf(pages)
	batch = resolveDraftPaths(batch, rows.byWPID)
	if len(batch) == 0 {
		return nil
	}

	r := reconciler{deps: deps, rows: rows, state: state, siteID: owner.ID, now: deps.now()}
	taken := make([]content.Finding, 0)
	apply := func(c context.Context) error {
		taken = taken[:0]
		waiting := waitingProducts(r.rows.byPath)
		for i := range batch {
			current, known, foreign := match(batch[i], r.rows)
			if foreign {
				taken = append(taken, pathTaken(batch[i], current))
				continue
			}
			if !known && batch[i].Type == pagemap.WPProduct && waiting > 0 {
				current, known = pagemap.ClaimProduct(storeCandidate(batch[i]), slices.Collect(maps.Values(r.rows.byPath)))
				if known {
					waiting--
				}
			}
			if keepErr := r.keep(c, batch[i], current, known); keepErr != nil {
				return keepErr
			}
		}
		return nil
	}

	if applyErr := deps.inUnit(ctx, apply); applyErr != nil {
		return applyErr
	}
	state.Findings = append(state.Findings, taken...)
	return nil
}

func (r reconciler) keep(ctx context.Context, item pulledItem, current pagemap.Page, known bool) error {
	next, drifted := merge(current, known, item, r.siteID, r.now)
	if known && current.Path != next.Path {
		delete(r.rows.byPath, current.Path)
	}

	if known {
		if err := r.deps.Pages.Update(ctx, next); err != nil {
			return err
		}
		r.state.Updated++
	} else {
		if err := r.deps.Pages.Insert(ctx, next); err != nil {
			return err
		}
		r.state.Created++
	}
	if drifted {
		r.state.Drifted++
	}
	r.rows.byPath[next.Path] = next
	r.rows.byWPID[keyOf(next.WPType, *next.WPID)] = next

	ours, err := generatedTargets(ctx, r.deps, next.ID, known)
	if err != nil {
		return err
	}
	return r.deps.Links.ReplaceForPage(ctx, next.ID, pulledLinks(next, r.rows.byPath, item.Links, r.now, ours))
}

func resolveDraftPaths(batch []pulledItem, byWPID map[siteKey]pagemap.Page) []pulledItem {
	known := make(map[siteKey]string, len(byWPID)+len(batch))
	for key := range byWPID {
		known[key] = byWPID[key].Path
	}
	for i := range batch {
		if batch[i].Path != "" {
			known[keyOf(batch[i].Type, batch[i].WPID)] = batch[i].Path
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

func draftPath(item pulledItem, known map[siteKey]string) (string, bool) {
	if item.Slug == "" {
		return "", false
	}

	base := "/"
	if item.ParentWPID != 0 {
		parent, ok := known[keyOf(item.Type, item.ParentWPID)]
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

func match(item pulledItem, rows siteRows) (page pagemap.Page, known, foreign bool) {
	if numbered, ok := rows.byWPID[keyOf(item.Type, item.WPID)]; ok {
		return numbered, true, false
	}
	page, ok := rows.byPath[item.Path]
	switch {
	case !ok:
		return pagemap.Page{}, false, false
	case !page.WPType.SameFamily(item.mappedType()):
		return page, false, true
	default:
		return page, true, false
	}
}

func waitingProducts(byPath map[string]pagemap.Page) int {
	count := 0
	for path := range byPath {
		if byPath[path].WPType == pagemap.WPProduct && byPath[path].WPID == nil {
			count++
		}
	}
	return count
}

func storeCandidate(item pulledItem) pagemap.Page {
	wpID := item.WPID
	return pagemap.Page{
		Path: item.Path, Slug: pagemap.Slug(item.Path), WPType: pagemap.WPProduct, WPID: &wpID,
		Observed: pagemap.Observed{Slug: item.Slug, Title: item.Title},
	}
}

func pathTaken(item pulledItem, row pagemap.Page) content.Finding {
	found := item.mappedType()
	return content.Finding{
		Severity: content.SeverityWarn,
		Code:     CodePathTakenOnSite,
		Message: "the site holds a " + string(found) + " at " + item.Path + " and the page map plans a " +
			string(row.WPType) + " there, so the two were kept apart; change the row's type or its path",
		Details: map[string]any{
			"pageId": row.ID, "path": item.Path, "planned": string(row.WPType),
			"found": string(found), "wpId": item.WPID,
		},
	}
}

func merge(current pagemap.Page, known bool, item pulledItem, siteID string,
	now time.Time) (next pagemap.Page, drifted bool) {
	next = current
	if !known {
		next = pagemap.Page{ID: id.New(), SiteID: siteID, CreatedAt: now}
	}

	drifted = known && next.ContentHash != "" && next.ContentHash != item.ContentHash

	if !hasPlan(current, known) {
		next.Path = item.Path
		next.Slug = pagemap.Slug(item.Path)
		next.Title = orKept(item.Title, next.Title)
		next.H1 = orKept(item.H1, next.H1)
		next.MetaTitle = orKept(item.Meta.Title, next.MetaTitle)
		next.MetaDescription = orKept(item.Meta.Description, next.MetaDescription)
		next.Canonical = orKept(item.Meta.Canonical, next.Canonical)
		next.Status = pagemap.StatusFromWordPress(item.Status)
	}
	next.WPType = item.mappedType()
	if known && next.WPType.StoreAddressed() && next.Path != item.Path {
		if current.WPID == nil && next.PlannedPath == "" {
			next.PlannedPath = current.Path
			next.Status = pagemap.StatusFromWordPress(item.Status)
		}
		next.Path = item.Path
		next.Slug = pagemap.Slug(item.Path)
	}
	if next.PlannedPath == next.Path {
		next.PlannedPath = ""
	}
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

func hasPlan(current pagemap.Page, known bool) bool {
	switch {
	case !known:
		return false
	case current.Status == pagemap.StatusPlanned, current.ContentHash != "":
		return true
	default:
		return current.Path != current.Observed.Path() ||
			current.Title != current.Observed.Title ||
			current.H1 != current.Observed.H1
	}
}

func orKept(reported, stored string) string {
	if reported == "" {
		return stored
	}
	return reported
}

func generatedTargets(ctx context.Context, deps Deps, pageID string, known bool) (map[string]struct{}, error) {
	if !known {
		return nil, nil
	}

	stored, err := deps.Links.ListForPage(ctx, pageID)
	if err != nil {
		return nil, err
	}
	ours := make(map[string]struct{}, len(stored))
	for i := range stored {
		if stored[i].Origin == pagemap.OriginGenerated {
			ours[stored[i].ToURL] = struct{}{}
		}
	}
	return ours, nil
}

func pulledLinks(page pagemap.Page, byPath map[string]pagemap.Page, links []wp.ContentLink, at time.Time,
	ours map[string]struct{}) []pagemap.PageLink {
	out := make([]pagemap.PageLink, 0, len(links))
	for i := range links {
		path, err := pagemap.NormalizePath(links[i].Href)
		if err != nil {
			continue
		}

		origin := pagemap.OriginObserved
		if _, generated := ours[path]; generated {
			origin = pagemap.OriginGenerated
		}
		link := pagemap.PageLink{
			ID: id.New(), SiteID: page.SiteID, FromPageID: page.ID, ToURL: path,
			AnchorText: links[i].Anchor, Origin: origin, ObservedAt: at,
		}
		if target, ok := byPath[path]; ok {
			link.ToPageID = &target.ID
		}
		if built, ok := observedLink(link); ok {
			out = append(out, built)
		}
	}
	return out
}
