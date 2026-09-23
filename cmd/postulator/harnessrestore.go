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
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/domain/run"
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
	if len(listed.Items) == 0 {
		return nil
	}
	return placeOnSite(ctx, core, site, listed.Items[0].ID)
}

func placeOnSite(ctx context.Context, core *app.Core, site *wptest.Server, siteID string) error {
	mapped, err := allPages(ctx, core, siteID)
	if err != nil {
		return err
	}
	bodies, err := generatedBodies(ctx, core, siteID)
	if err != nil {
		return err
	}

	sort.SliceStable(mapped, func(a, b int) bool {
		return depthOf(mapped[a].Path) < depthOf(mapped[b].Path)
	})

	wpByPath := make(map[string]int64, len(mapped))
	for index := range mapped {
		page := &mapped[index]
		if page.Status == statusArchived || page.Status == statusPlanned {
			continue
		}

		item := wptest.Item{
			Type: page.WPType, Title: page.Title, H1: page.H1, Slug: slugOf(page.Path),
			Content: bodies[page.ID], Status: wpStatus(page.Status), Modified: time.Now().UTC(),
			Parent: wpByPath[parentOf(page.Path)],
		}
		if page.WPID != nil {
			item.ID = *page.WPID
		}
		wpByPath[page.Path] = site.Restore(item)[0].ID
	}
	return nil
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

func generatedBodies(ctx context.Context, core *app.Core, siteID string) (map[string]string, error) {
	bodies := make(map[string]string, len(completedRun))

	listed, err := core.Runs.List(ctx, runs.ListRequest{
		SiteID: siteID, ListRequest: dto.ListRequest{Limit: 20},
	})
	if err != nil {
		return nil, err
	}

	for index := range listed.Items {
		items, itemsErr := core.Runs.ListItems(ctx, runs.ListItemsRequest{
			RunID: listed.Items[index].ID, ListRequest: dto.ListRequest{Limit: 50},
		})
		if itemsErr != nil {
			return nil, itemsErr
		}
		for item := range items.Items {
			target := items.Items[item]
			held, getErr := core.Runs.GetArtifact(ctx, runs.GetArtifactRequest{
				ItemID: target.ID, Kind: string(run.ArtifactBodyHTML),
			})
			if getErr != nil {
				continue
			}
			if held.Artifact.Content != "" {
				bodies[target.TargetID] = held.Artifact.Content
			}
		}
	}
	return bodies, nil
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
