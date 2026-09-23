package wpmedia_test

import (
	"context"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/images"
	"github.com/davidmovas/postulator/internal/adapters/images/wpmedia"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type oneSite struct {
	client *wp.Client
	err    error
}

func (o oneSite) Client(context.Context, string) (*wp.Client, error) {
	return o.client, o.err
}

func newSource(t *testing.T, server *wptest.Server) *wpmedia.Source {
	t.Helper()

	client, err := wp.New(wp.Config{
		BaseURL:       server.URL(),
		Username:      wptest.DefaultUser,
		AppPassword:   wptest.DefaultPassword,
		AllowInsecure: true,
	}, wp.WithRateLimit(0), wp.WithBackoff(func(int) time.Duration { return 0 }))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return wpmedia.New(oneSite{client: client})
}

func TestPickSearchesTheMediaLibrary(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	source := newSource(t, server)

	client, err := wp.New(wp.Config{
		BaseURL:       server.URL(),
		Username:      wptest.DefaultUser,
		AppPassword:   wptest.DefaultPassword,
		AllowInsecure: true,
	}, wp.WithRateLimit(0))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for _, name := range []string{"espresso-cup.png", "kettle.png"} {
		if _, uploadErr := client.UploadMedia(t.Context(), wp.Media{
			Filename: name, ContentType: "image/png", Bytes: []byte{0x89}, Alt: "a " + name,
		}); uploadErr != nil {
			t.Fatalf("UploadMedia: %v", uploadErr)
		}
	}

	picked, err := source.Pick(t.Context(), images.Query{SiteID: "site", Term: "espresso"})
	if err != nil {
		t.Fatalf("Pick: %v", err)
	}
	if len(picked) != 1 {
		t.Fatalf("Pick returned %d images, want 1: %+v", len(picked), picked)
	}
	if picked[0].WPID == 0 || picked[0].URL == "" || picked[0].Alt != "a espresso-cup.png" {
		t.Fatalf("picked = %+v", picked[0])
	}
	if len(picked[0].Bytes) != 0 {
		t.Error("a library pick carries no bytes, only an id and a url")
	}
}

func TestPickFallsBackToTheTitleForAlternativeText(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	source := newSource(t, server)

	client, newErr := wp.New(wp.Config{
		BaseURL: server.URL(), Username: wptest.DefaultUser,
		AppPassword: wptest.DefaultPassword, AllowInsecure: true,
	}, wp.WithRateLimit(0))
	if newErr != nil {
		t.Fatalf("New: %v", newErr)
	}
	if _, err := client.UploadMedia(t.Context(), wp.Media{
		Filename: "espresso.png", ContentType: "image/png", Bytes: []byte{0x89},
	}); err != nil {
		t.Fatalf("UploadMedia: %v", err)
	}

	picked, err := source.Pick(t.Context(), images.Query{SiteID: "site", Term: "espresso", Limit: 5})
	if err != nil {
		t.Fatalf("Pick: %v", err)
	}
	if len(picked) != 1 || picked[0].Alt != "espresso.png" {
		t.Fatalf("picked = %+v", picked)
	}
}

func TestPickReportsWhatItCannotDo(t *testing.T) {
	t.Parallel()

	boom := errors.New(errors.NotFound, "no such site")

	if _, err := wpmedia.New(oneSite{}).Pick(t.Context(), images.Query{Term: "x"}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
	if _, err := wpmedia.New(oneSite{err: boom}).Pick(t.Context(), images.Query{SiteID: "site"}); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("code = %q, want %q", errors.CodeOf(err), errors.NotFound)
	}
}
