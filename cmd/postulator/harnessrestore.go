//go:build uiharness

package main

import (
	"context"
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

const restorePageLimit = 100

func repopulate(ctx context.Context, core *app.Core, site *wptest.Server) error {
	listed, err := core.Sites.List(ctx, sites.ListRequest{ListRequest: dto.ListRequest{Limit: 10}})
	if err != nil {
		return err
	}
	if len(listed.Items) == 0 {
		return nil
	}
	siteID := listed.Items[0].ID

	mapped, err := allPages(ctx, core, siteID)
	if err != nil {
		return err
	}
	bodies, err := generatedBodies(ctx, core, siteID)
	if err != nil {
		return err
	}

	sort.SliceStable(mapped, func(a, b int) bool {
		return strings.Count(mapped[a].Path, "/") < strings.Count(mapped[b].Path, "/")
	})

	wpByPage := make(map[string]int64, len(mapped))
	for index := range mapped {
		page := &mapped[index]
		if page.Status == statusArchived {
			continue
		}

		item := wptest.Item{
			Type: page.WPType, Title: page.Title, H1: page.H1, Slug: slugOf(page.Path),
			Content: bodies[page.ID], Status: wpStatus(page.Status), Modified: time.Now().UTC(),
		}
		if page.WPID != nil {
			item.ID = *page.WPID
		}
		if page.ParentPageID != nil {
			item.Parent = wpByPage[*page.ParentPageID]
		}
		wpByPage[page.ID] = site.Restore(item)[0].ID
	}
	return nil
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
