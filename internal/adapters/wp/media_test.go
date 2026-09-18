package wp_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestUploadMediaTakesTwoRequestsToSetAlternativeText(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	uploaded, err := newClient(t, server).UploadMedia(t.Context(), wp.Media{
		Filename:    "koffein.png",
		ContentType: "image/png",
		Bytes:       []byte{0x89, 0x50, 0x4e, 0x47},
		Alt:         "Koffein powder",
		Title:       "Koffein",
	})
	if err != nil {
		t.Fatalf("UploadMedia: %v", err)
	}

	if uploaded.ID == 0 || uploaded.SourceURL == "" || uploaded.MimeType != "image/png" {
		t.Errorf("uploaded = %+v", uploaded)
	}
	if uploaded.Alt != "Koffein powder" || uploaded.Title != "Koffein" {
		t.Errorf("uploaded = %+v, want the attributes from the second call", uploaded)
	}

	recorded := server.Requests()
	if len(recorded) != 2 {
		t.Fatalf("the client made %d requests, want an upload and an attribute write", len(recorded))
	}
	if got := recorded[0].Header.Get("Content-Disposition"); got != `attachment; filename="koffein.png"` {
		t.Errorf("Content-Disposition = %q", got)
	}
	if got := recorded[0].Header.Get("Content-Type"); got != "image/png" {
		t.Errorf("Content-Type = %q", got)
	}
	if len(recorded[0].Body) != 4 {
		t.Errorf("the upload carried %d bytes, want 4", len(recorded[0].Body))
	}
}

func TestUploadMediaSkipsTheSecondRequestWhenThereIsNothingToSet(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	if _, err := newClient(t, server).UploadMedia(t.Context(), wp.Media{
		Filename:    "koffein.png",
		ContentType: "image/png",
		Bytes:       []byte{0x89},
	}); err != nil {
		t.Fatalf("UploadMedia: %v", err)
	}
	if got := len(server.Requests()); got != 1 {
		t.Errorf("the client made %d requests, want 1", got)
	}
}

func TestUploadMediaRefusesAnEmptyUpload(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	client := newClient(t, server)

	cases := []struct {
		name  string
		media wp.Media
	}{
		{name: "no bytes", media: wp.Media{Filename: "koffein.png", ContentType: "image/png"}},
		{name: "no file name", media: wp.Media{ContentType: "image/png", Bytes: []byte{0x89}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := client.UploadMedia(t.Context(), tc.media); !errors.IsCode(err, errors.Invalid) {
				t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
			}
		})
	}
}

func TestCategoriesAreListedAndCreated(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.SeedCategory(wptest.Category{Name: "Koffein", Description: "the hub"})
	client := newClient(t, server)

	listed, err := client.ListCategories(t.Context(), wp.ListQuery{})
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	if listed.Total != 1 || len(listed.Items) != 1 || listed.Items[0].Name != "Koffein" {
		t.Fatalf("listed = %+v", listed)
	}

	created, err := client.CreateCategory(t.Context(), wp.CreateCategory{Name: "Koffein", Description: "a second one"})
	if err != nil {
		t.Fatalf("CreateCategory: %v", err)
	}
	if created.Slug != "koffein-2" {
		t.Errorf("slug = %q, want koffein-2", created.Slug)
	}
	if created.Description != "a second one" {
		t.Errorf("description = %q", created.Description)
	}

	if _, err = client.CreateCategory(t.Context(), wp.CreateCategory{}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
}

func TestListMediaSearchesTheLibrary(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	client := newClient(t, server)

	for _, name := range []string{"espresso-cup.png", "kettle.png", "espresso-machine.png"} {
		if _, err := client.UploadMedia(t.Context(), wp.Media{
			Filename: name, ContentType: "image/png", Bytes: []byte{0x89}, Alt: name,
		}); err != nil {
			t.Fatalf("UploadMedia %s: %v", name, err)
		}
	}

	cases := []struct {
		name   string
		query  wp.MediaQuery
		want   int
		total  int
		remain bool
	}{
		{name: "everything", query: wp.MediaQuery{}, want: 3, total: 3},
		{name: "by term", query: wp.MediaQuery{Search: "espresso"}, want: 2, total: 2},
		{name: "nothing matches", query: wp.MediaQuery{Search: "tea"}, want: 0},
		{name: "one per page", query: wp.MediaQuery{PerPage: 1}, want: 1, total: 3, remain: true},
		{name: "past the end", query: wp.MediaQuery{PerPage: 1, Page: 9}, want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := client.ListMedia(t.Context(), tc.query)
			if err != nil {
				t.Fatalf("ListMedia: %v", err)
			}
			if len(got.Items) != tc.want {
				t.Fatalf("ListMedia returned %d items, want %d", len(got.Items), tc.want)
			}
			if tc.total != 0 && got.Total != tc.total {
				t.Errorf("Total = %d, want %d", got.Total, tc.total)
			}
			if got.HasMore != tc.remain {
				t.Errorf("HasMore = %t, want %t", got.HasMore, tc.remain)
			}
			for _, item := range got.Items {
				if item.ID == 0 || item.SourceURL == "" {
					t.Errorf("item = %+v", item)
				}
			}
		})
	}
}

func TestListItemsFiltersBySlug(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(
		wptest.Item{Type: wptest.TypePage, Title: "Espresso"},
		wptest.Item{Type: wptest.TypePage, Title: "Kettle"},
	)

	got, err := newClient(t, server).ListItems(t.Context(), wp.TypePage, wp.ListQuery{Slug: "espresso"})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].Slug != "espresso" {
		t.Fatalf("ListItems by slug = %+v", got.Items)
	}
}
