package observe

import (
	"time"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

const CodePathTakenOnSite = "path_taken_on_site"

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

func (r *siteRows) remember(page pagemap.Page) {
	r.byPath[page.Path] = page
	if page.WPID != nil {
		r.byWPID[keyOf(page.WPType, *page.WPID)] = page
	}
}

func resolveDraftPaths(batch []item, byWPID map[siteKey]pagemap.Page) []item {
	known := make(map[siteKey]string, len(byWPID)+len(batch))
	for key := range byWPID {
		known[key] = byWPID[key].Path
	}
	for i := range batch {
		if batch[i].Path != "" {
			known[keyOf(batch[i].Type, batch[i].WPID)] = batch[i].Path
		}
	}

	out := make([]item, 0, len(batch))
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

func draftPath(it item, known map[siteKey]string) (string, bool) {
	if it.Slug == "" {
		return "", false
	}
	base := "/"
	if it.ParentWPID != 0 {
		parent, ok := known[keyOf(it.Type, it.ParentWPID)]
		if !ok {
			return "", false
		}
		base = parent
	}
	path, err := pagemap.NormalizePath(base + it.Slug + "/")
	if err != nil {
		return "", false
	}
	return path, true
}

func match(it item, rows siteRows) (page pagemap.Page, known, foreign bool) {
	if numbered, ok := rows.byWPID[keyOf(it.Type, it.WPID)]; ok {
		return numbered, true, false
	}
	page, ok := rows.byPath[it.Path]
	switch {
	case !ok:
		return pagemap.Page{}, false, false
	case !page.WPType.SameFamily(it.mappedType()):
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

func storeCandidate(it item) pagemap.Page {
	wpID := it.WPID
	return pagemap.Page{
		Path: it.Path, Slug: orKept(it.Slug, pagemap.Slug(it.Path)), WPType: pagemap.WPProduct, WPID: &wpID, Title: it.Title,
	}
}

type lastSeen struct {
	path     string
	title    string
	h1       string
	observed bool
	wrote    bool
}

func seenOf(identity pagemap.Identity, held bool, values []field.Value, blocks bool) lastSeen {
	seen := lastSeen{observed: held, wrote: blocks}
	if held && !queryPermalink(identity.URL) {
		if path, err := pagemap.NormalizePath(identity.URL); err == nil {
			seen.path = path
		}
	}
	for i := range values {
		seen.wrote = seen.wrote || values[i].WrittenRunID != nil
		switch values[i].Key {
		case field.Title:
			seen.title = values[i].Current
		case field.H1:
			seen.h1 = values[i].Current
		}
	}
	return seen
}

func (s lastSeen) movedFrom(path string) bool {
	return s.path != "" && s.path != path
}

func pathTaken(it item, row pagemap.Page) content.Finding {
	found := it.mappedType()
	return content.Finding{
		Severity: content.SeverityWarn,
		Code:     CodePathTakenOnSite,
		Message: "the site holds a " + string(found) + " at " + it.Path + " and the page map plans a " +
			string(row.WPType) + " there, so the two were kept apart; change the row's type or its path",
		Details: map[string]any{
			"pageId": row.ID, "path": it.Path, "planned": string(row.WPType),
			"found": string(found), "wpId": it.WPID,
		},
	}
}

func merge(current pagemap.Page, known, planned bool, it item, siteID string, now time.Time) pagemap.Page {
	next := current
	if !known {
		next = pagemap.Page{ID: id.New(), SiteID: siteID, CreatedAt: now}
	}

	if !planned {
		next.Path = it.Path
		next.Slug = orKept(it.Slug, pagemap.Slug(it.Path))
		next.Title = orKept(it.Title, next.Title)
		next.H1 = orKept(it.H1, next.H1)
		next.MetaTitle = orKept(it.Meta.Title, next.MetaTitle)
		next.MetaDescription = orKept(it.Meta.Description, next.MetaDescription)
		next.Canonical = orKept(it.Meta.Canonical, next.Canonical)
		next.Status = pagemap.StatusFromWordPress(it.Status)
	}
	next.WPType = it.mappedType()
	if known && next.WPType.StoreAddressed() && next.Path != it.Path {
		if current.WPID == nil && next.PlannedPath == "" {
			next.PlannedPath = current.Path
			next.Status = pagemap.StatusFromWordPress(it.Status)
		}
		next.Path = it.Path
		next.Slug = orKept(it.Slug, pagemap.Slug(it.Path))
	}
	if next.PlannedPath == next.Path {
		next.PlannedPath = ""
	}
	if next.WPID == nil && next.Status == pagemap.StatusPlanned {
		next.Status = pagemap.StatusFromWordPress(it.Status)
	}
	next.WPID = &it.WPID
	next.LastSyncedAt = &now
	next.UpdatedAt = now
	if !it.Modified.IsZero() {
		modified := it.Modified.UTC()
		next.WPModifiedAt = &modified
	}
	return next
}

func hasPlan(current pagemap.Page, known bool, seen lastSeen) bool {
	switch {
	case !known:
		return false
	case current.Status == pagemap.StatusPlanned, seen.wrote, !seen.observed:
		return true
	default:
		return seen.movedFrom(current.Path) || current.Title != seen.title || current.H1 != seen.h1
	}
}

func orKept(reported, stored string) string {
	if reported == "" {
		return stored
	}
	return reported
}
