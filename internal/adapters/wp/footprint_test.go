package wp_test

import (
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

const programName = "postulator"

func TestNothingTheClientSendsNamesTheProgram(t *testing.T) {
	t.Parallel()

	const user, password = "editor", "wxyz WXYZ 9876 abcd"

	server := wptest.New(t, wptest.WithCredentials(user, password))
	seeded := server.Seed(
		wptest.Item{Type: wptest.TypePage, Title: "Coffee", Slug: "coffee", Status: "publish", Content: "<p>Beans.</p>"},
		wptest.Item{Type: wptest.TypeProduct, Title: "Powder", Slug: "powder", Status: "publish", Content: "<p>Made by hand.</p>"},
		wptest.Item{Type: wptest.TypePage, Title: "Draft", Slug: "draft", Status: "draft", Content: "<p>Soon.</p>"},
	)
	page, product, draft := seeded[0], seeded[1], seeded[2]

	client, err := wp.New(wp.Config{BaseURL: server.URL(), Username: user, AppPassword: password, AllowInsecure: true},
		wp.WithRateLimit(0), wp.WithBackoff(func(int) time.Duration { return 0 }))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := t.Context()
	title, body := "Fresh coffee", "<p>Roasted today.</p>"
	calls := []struct {
		name string
		call func() error
	}{
		{name: "Probe", call: func() error { _, err := client.Probe(ctx); return err }},
		{name: "Capabilities", call: func() error { _, err := client.Capabilities(ctx); return err }},
		{name: "ListContent", call: func() error { _, err := client.ListContent(ctx, wp.ContentQuery{}); return err }},
		{name: "ListItems", call: func() error { _, err := client.ListItems(ctx, wp.TypePage, wp.ListQuery{}); return err }},
		{name: "GetItem", call: func() error { _, err := client.GetItem(ctx, wp.TypePage, page.ID); return err }},
		{name: "CreateItem", call: func() error {
			_, err := client.CreateItem(ctx, wp.TypePage, wp.CreateItem{Title: title, Content: body, Slug: "fresh-coffee"})
			return err
		}},
		{name: "UpdateItem", call: func() error {
			_, err := client.UpdateItem(ctx, wp.TypePage, page.ID, wp.UpdateItem{Title: &title, Content: &body})
			return err
		}},
		{name: "GetRaw", call: func() error { _, err := client.GetRaw(ctx, wp.TypePage, page.ID); return err }},
		{name: "PutRaw", call: func() error { _, err := client.PutRaw(ctx, wp.TypePage, page.ID, body, ""); return err }},
		{name: "SetSEOMeta", call: func() error {
			_, err := client.SetSEOMeta(ctx, wp.TypePage, page.ID, wp.SEOMeta{Title: title, Description: "Beans, roasted."})
			return err
		}},
		{name: "GetSEOMeta", call: func() error { _, err := client.GetSEOMeta(ctx, wp.TypePage, page.ID); return err }},
		{name: "ReplaceSEOMeta", call: func() error {
			_, err := client.ReplaceSEOMeta(ctx, wp.TypePage, page.ID, wp.SEOMeta{Title: title}, []string{"title"})
			return err
		}},
		{name: "PreviewLink", call: func() error { _, err := client.PreviewLink(ctx, wp.TypePage, draft.ID); return err }},
		{name: "GetRendered", call: func() error { _, err := client.GetRendered(ctx, wp.TypePage, page.ID); return err }},
		{name: "GetMeta", call: func() error { _, err := client.GetMeta(ctx, wp.TypePage, page.ID, []string{"color"}); return err }},
		{name: "UploadMedia", call: func() error {
			_, err := client.UploadMedia(ctx, wp.Media{Filename: "coffee-beans.png", ContentType: "image/png", Bytes: []byte("\x89PNG"), Alt: "Roasted beans", Title: "Beans"})
			return err
		}},
		{name: "ListMedia", call: func() error { _, err := client.ListMedia(ctx, wp.MediaQuery{}); return err }},
		{name: "GetProduct", call: func() error { _, err := client.GetProduct(ctx, product.ID); return err }},
		{name: "UpdateProduct", call: func() error {
			short := "Ground fine."
			_, err := client.UpdateProduct(ctx, product.ID, wp.UpdateProduct{ShortDescription: &short})
			return err
		}},
		{name: "Commerce", call: func() error { _, err := client.Commerce(ctx); return err }},
		{name: "Visit", call: func() error { _, err := client.Visit(ctx, server.URL()+"/product/powder/"); return err }},
	}
	for _, step := range calls {
		if err := step.call(); err != nil {
			t.Fatalf("%s: %v", step.name, err)
		}
	}

	for _, recorded := range server.Requests() {
		where := recorded.Method + " " + recorded.Path
		for key, values := range recorded.Header {
			for _, value := range values {
				if namesTheProgram(value) {
					t.Errorf("%s: header %s = %q names the program", where, key, value)
				}
			}
		}
		for key, values := range recorded.Query {
			for _, value := range values {
				if namesTheProgram(value) {
					t.Errorf("%s: query %s = %q names the program", where, key, value)
				}
			}
		}
		if namesTheProgram(string(recorded.Body)) {
			t.Errorf("%s: the body names the program: %q", where, recorded.Body)
		}
	}
	for _, upload := range server.Uploads() {
		for field, value := range map[string]string{"filename": upload.Filename, "title": upload.Title, "alt": upload.Alt} {
			if namesTheProgram(value) {
				t.Errorf("upload %s = %q names the program", field, value)
			}
		}
	}
}

func namesTheProgram(value string) bool {
	return strings.Contains(strings.ToLower(value), programName)
}
