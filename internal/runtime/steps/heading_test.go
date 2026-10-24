package steps_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
)

const publishedWithoutH1 = `<p>Espresso is a way to make <a href="/coffee/">coffee</a>.</p>`

func TestACreatedPageCarriesNoH1WhereTheThemePrintsTheTitle(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	sc := publishContext(t)
	deps.Pictures = &pictureStub{owesH1: map[string]bool{sc.Page.ID: false}}

	published := runPublish(t, deps, sc)
	stored, _ := server.Lookup(published.WPID)
	if !published.Created || stored.Content != publishedWithoutH1 {
		t.Errorf("the created page holds %q, want our body without its H1, the theme printing the title", stored.Content)
	}
}

func TestACreationWithNoObservedRenderCarriesNoH1(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	published := runPublish(t, deps, publishContext(t))
	stored, _ := server.Lookup(published.WPID)
	if stored.Content != publishedWithoutH1 {
		t.Errorf("the created page holds %q, want no H1 while nothing says the theme prints none", stored.Content)
	}
}

func TestOurH1OnlyWhereTheThemePrintsNone(t *testing.T) {
	t.Parallel()

	t.Run("a page we create where the theme prints none", func(t *testing.T) {
		t.Parallel()

		deps, server := imageDeps(t)
		sc := publishContext(t)
		deps.Pictures = &pictureStub{owesH1: map[string]bool{sc.Page.ID: true}}

		published := runPublish(t, deps, sc)
		if stored, _ := server.Lookup(published.WPID); stored.Content != publishBody {
			t.Errorf("the created page holds %q, want our body with its H1", stored.Content)
		}
	})

	t.Run("an existing page with no text of the client's, where the theme prints none", func(t *testing.T) {
		t.Parallel()

		kept := &content.KeptBody{Raw: "", Hash: block.BodyHash("")}
		deps, server, _, sc := keptRunOver(t, "", oursBody, kept)
		deps.Pictures = &pictureStub{owesH1: map[string]bool{sc.Page.ID: true}}

		published := runPublish(t, deps, sc)
		if stored, _ := server.Lookup(published.WPID); stored.Content != "<h1>Espresso</h1>"+oursBody {
			t.Errorf("the site holds %q, want our body with its H1, nothing on the page printing one", stored.Content)
		}
	})

	t.Run("an existing page whose own text already carries an H1", func(t *testing.T) {
		t.Parallel()

		clients := "<h1>Lab guides</h1>\n\n" + clientText
		kept := &content.KeptBody{Raw: clients, Hash: block.BodyHash(clients)}
		deps, server, _, sc := keptRunOver(t, clients, oursBody, kept)
		deps.Pictures = &pictureStub{owesH1: map[string]bool{sc.Page.ID: true}}

		published := runPublish(t, deps, sc)
		if stored, _ := server.Lookup(published.WPID); stored.Content != clients+"\n\n"+oursBody {
			t.Errorf("the site holds %q, want the client's heading and text kept and ours after them without a second H1", stored.Content)
		}
	})
}
