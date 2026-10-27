//go:build uiharness

package main

import (
	"context"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

const (
	restorePageLimit = 100

	unlockPoll = 500 * time.Millisecond
	unlockWait = time.Hour
)

func restoreWhenReady(ctx context.Context, core *app.Core, site *wptest.Server) error {
	if !core.Locked() {
		return repopulate(ctx, core, site)
	}

	log.Print("the harness holds the fake site back until the core is unlocked")
	go awaitUnlock(context.WithoutCancel(ctx), core, site)
	return nil
}

func awaitUnlock(ctx context.Context, core *app.Core, site *wptest.Server) {
	deadline := time.Now().Add(unlockWait)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return
		case <-time.After(unlockPoll):
		}

		if core.Locked() {
			continue
		}
		if err := repopulate(ctx, core, site); err != nil {
			log.Printf("the harness could not restore the fake site: %v", err)
		}
		return
	}
	log.Print("the harness gave up waiting for the core to be unlocked; the fake site stays empty")
}

func repopulate(ctx context.Context, core *app.Core, site *wptest.Server) error {
	if core.Locked() {
		return nil
	}

	listed, err := core.Sites.List(ctx, sites.ListRequest{ListRequest: dto.ListRequest{Limit: 10}})
	if err != nil {
		return err
	}
	for i := range listed.Items {
		if listed.Items[i].Name == siteName {
			return placeOnSite(ctx, core, site, listed.Items[i].ID)
		}
	}
	return nil
}

func placeOnSite(ctx context.Context, core *app.Core, site *wptest.Server, siteID string) error {
	mapped, err := allPages(ctx, core, siteID)
	if err != nil {
		return err
	}

	sort.SliceStable(mapped, func(a, b int) bool {
		return depthOf(mapped[a].Path) < depthOf(mapped[b].Path)
	})

	titles := make(map[string]string, len(mapped))
	for index := range mapped {
		if mapped[index].Status != statusArchived && mapped[index].Status != statusPlanned {
			titles[mapped[index].Path] = mapped[index].Title
		}
	}

	wpByPath := make(map[string]int64, len(mapped))
	for index := range mapped {
		page := &mapped[index]
		if page.Status == statusArchived || page.Status == statusPlanned {
			continue
		}

		body, bodyErr := heldBody(ctx, core, page)
		if bodyErr != nil {
			return bodyErr
		}
		item := wptest.Item{
			Type: page.WPType, Title: page.Title, H1: page.H1, Slug: slugOf(page.Path),
			Content: body, Status: wpStatus(page.Status), Modified: time.Now().UTC(),
			Parent: wpByPath[parentOf(page.Path)], Crumbs: crumbsOf(page.Path, page.Title, titles),
			Meta: customMetaOf(page.Path), Terms: storeTermsOf(page.Path),
		}
		if page.WPID != nil {
			item.ID = *page.WPID
		}
		wpByPath[page.Path] = site.Restore(item)[0].ID
	}
	return nil
}

func customMetaOf(path string) map[string]string {
	subtitle, held := seedSubtitles()[path]
	if !held {
		return nil
	}
	return map[string]string{subtitleKey: subtitle}
}

func crumbsOf(path, title string, titles map[string]string) []wptest.Crumb {
	if _, nested := titles[parentOf(path)]; !nested {
		return nil
	}
	ancestors := make([]wptest.Crumb, 0)
	for ancestor := parentOf(path); ancestor != "" && ancestor != "/"; ancestor = parentOf(ancestor) {
		if name, known := titles[ancestor]; known {
			ancestors = append([]wptest.Crumb{{Name: name, Path: ancestor}}, ancestors...)
		}
	}
	crumbs := append([]wptest.Crumb{{Name: "Home", Path: "/"}}, ancestors...)
	return append(crumbs, wptest.Crumb{Name: title, Path: path})
}

func heldBody(ctx context.Context, core *app.Core, page *pages.Page) (string, error) {
	model, err := core.Observe.Page(ctx, page.ID)
	if err != nil {
		return "", err
	}
	if model.Body != nil {
		return model.Body.Raw, nil
	}
	return seedBodies()[page.Path], nil
}

func depthOf(path string) int {
	return strings.Count(strings.Trim(path, "/"), "/")
}

func parentOf(path string) string {
	trimmed := strings.Trim(path, "/")
	cut := strings.LastIndex(trimmed, "/")
	if cut < 0 {
		return ""
	}
	return "/" + trimmed[:cut] + "/"
}

func allPages(ctx context.Context, core *app.Core, siteID string) ([]pages.Page, error) {
	out := make([]pages.Page, 0, restorePageLimit)
	cursor := ""

	for {
		page, err := core.Pages.List(ctx, pages.ListRequest{
			SiteID: siteID, ListRequest: dto.ListRequest{Limit: restorePageLimit, Cursor: cursor},
		})
		if err != nil {
			return nil, err
		}
		out = append(out, page.Items...)
		if page.Next == "" {
			return out, nil
		}
		cursor = string(page.Next)
	}
}

func slugOf(path string) string {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	return segments[len(segments)-1]
}

func wpStatus(status string) string {
	if status == statusPublished {
		return "publish"
	}
	return "draft"
}
