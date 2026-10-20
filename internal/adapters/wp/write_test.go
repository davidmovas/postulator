package wp_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func pointerTo[T any](value T) *T {
	return &value
}

func TestCreateItemReturnsTheSlugWordPressActuallyUsed(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Powder"})
	server.ResetRequests()

	created, err := newClient(t, server).CreateItem(t.Context(), wp.TypePage, wp.CreateItem{
		Title:   "Powder",
		Content: "<p>x</p>",
		Slug:    "powder",
	})
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}

	if created.Slug != "powder-2" {
		t.Errorf("slug = %q, want powder-2", created.Slug)
	}
	if created.Link == "" {
		t.Error("the created item must carry the permalink WordPress computed")
	}
	if created.Status != "draft" {
		t.Errorf("status = %q, want the draft default", created.Status)
	}

	recorded := server.Requests()
	if len(recorded) != 2 {
		t.Fatalf("the client made %d requests, want a create and a re-read", len(recorded))
	}
	if recorded[0].Method != "POST" || recorded[1].Method != "GET" {
		t.Errorf("requests = %s %s", recorded[0].Method, recorded[1].Method)
	}
}

func TestCreateItemPlacesAChildUnderItsParent(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	parent := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein"})[0]

	created, err := newClient(t, server).CreateItem(t.Context(), wp.TypePage, wp.CreateItem{
		Title:  "Powder",
		Status: "publish",
		Parent: pointerTo(parent.ID),
	})
	if err != nil {
		t.Fatalf("CreateItem: %v", err)
	}
	if created.Parent != parent.ID {
		t.Errorf("parent = %d, want %d", created.Parent, parent.ID)
	}
	if created.Status != "publish" {
		t.Errorf("status = %q, want publish", created.Status)
	}
}

func TestAnUpdateCarriesOnlyWhatAnUpdateMayWrite(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	parent := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein"})[0]
	child := server.Seed(wptest.Item{
		Type: wptest.TypePage, Title: "Powder", Slug: "powder", Status: "publish", Parent: parent.ID, Categories: []int64{7},
	})[0]
	server.ResetRequests()

	updated, err := newClient(t, server).UpdateItem(t.Context(), wp.TypePage, child.ID, wp.UpdateItem{
		Title: pointerTo("Ground powder"), Content: pointerTo("<p>x</p>"), Excerpt: pointerTo("Short"), FeaturedMedia: pointerTo(int64(12)),
	})
	if err != nil {
		t.Fatalf("UpdateItem: %v", err)
	}
	if updated.Parent != parent.ID || updated.Slug != "powder" || updated.Status != "publish" || len(updated.Categories) != 1 {
		t.Errorf("the update moved what it may not write: %+v", updated)
	}
	request, _ := server.LastRequest()
	var sent map[string]any
	if err = json.Unmarshal(request.Body, &sent); err != nil {
		t.Fatalf("decode the update: %v", err)
	}
	for key := range sent {
		if !slices.Contains([]string{"title", "content", "excerpt", "featured_media"}, key) {
			t.Errorf("the update sent %s", key)
		}
	}
}

func TestUpdateItemRefusesAnEmptyChange(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Powder"})
	server.ResetRequests()

	_, err := newClient(t, server).UpdateItem(t.Context(), wp.TypePage, seeded[0].ID, wp.UpdateItem{})
	if !errors.IsCode(err, errors.Invalid) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
	if len(server.Requests()) != 0 {
		t.Error("an empty update must not reach the site")
	}
}

func TestDeleteItemTrashesUnlessItIsForced(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(
		wptest.Item{Type: wptest.TypePage, Title: "One"},
		wptest.Item{Type: wptest.TypePage, Title: "Two"},
	)
	client := newClient(t, server)

	if err := client.DeleteItem(t.Context(), wp.TypePage, seeded[0].ID, false); err != nil {
		t.Fatalf("DeleteItem: %v", err)
	}
	trashed, ok := server.Lookup(seeded[0].ID)
	if !ok || trashed.Status != "trash" {
		t.Errorf("item = %+v, %t", trashed, ok)
	}

	if err := client.DeleteItem(t.Context(), wp.TypePage, seeded[1].ID, true); err != nil {
		t.Fatalf("DeleteItem forced: %v", err)
	}
	if _, ok = server.Lookup(seeded[1].ID); ok {
		t.Error("a forced delete removes the item")
	}

	if err := client.DeleteItem(t.Context(), wp.TypePage, 404, true); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.NotFound)
	}
}

func TestTheGenericWriteMethodsCoverPagesAndPostsOnly(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	client := newClient(t, server)
	server.ResetRequests()

	if _, err := client.CreateItem(t.Context(), wp.TypeProduct, wp.CreateItem{Title: "x"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("create code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
	if _, err := client.UpdateItem(t.Context(), wp.TypeProduct, 1, wp.UpdateItem{Title: pointerTo("x")}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("update code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
	if err := client.DeleteItem(t.Context(), wp.TypeProductCategory, 1, true); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("delete code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
	if len(server.Requests()) != 0 {
		t.Error("a rejected write must not reach the site")
	}
}
