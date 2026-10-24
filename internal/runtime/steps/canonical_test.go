package steps_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/run"
)

func canonicalsSent(t *testing.T, server *wptest.Server) []string {
	t.Helper()

	out := make([]string, 0)
	for _, request := range writesOf(server) {
		if !strings.Contains(request.Path, "/seo-meta/") {
			continue
		}
		var sent map[string]json.RawMessage
		if err := json.Unmarshal(request.Body, &sent); err != nil {
			t.Fatalf("decode the SEO write %s: %v", request.Body, err)
		}
		if canonical, carries := sent["canonical"]; carries {
			out = append(out, strings.Trim(string(canonical), `"`))
		}
	}
	return out
}

func TestTheCanonicalIsNeverWrittenToAnExistingPage(t *testing.T) {
	t.Parallel()

	deps, server, sc := draftOverALivePage(t)
	deps.Types = typesStub{canonical: true}
	sc.Run.PublishMode = run.PublishLive
	server.ResetRequests()

	runPublish(t, deps, sc)
	if sent := canonicalsSent(t, server); len(sent) != 0 {
		t.Fatalf("an update sent the canonical %v, which belongs to the client's page", sent)
	}
}

func TestALiveCreationTakesItsPermalinkAsTheCanonical(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		enabled bool
	}{
		{name: "a scope that enables the canonical", enabled: true},
		{name: "a scope that leaves it off"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps, server := imageDeps(t)
			deps.Bodies = &bodyLog{}
			deps.Types = typesStub{canonical: tc.enabled}
			sc := publishContext(t)
			sc.Run.PublishMode = run.PublishLive
			server.ResetRequests()

			published := runPublish(t, deps, sc)
			sent := canonicalsSent(t, server)
			switch {
			case !tc.enabled && len(sent) != 0:
				t.Fatalf("a scope with the canonical off sent %v", sent)
			case tc.enabled && (len(sent) != 1 || sent[0] != published.URL):
				t.Fatalf("the creation sent the canonical %v, want only the permalink WordPress answered, %q", sent, published.URL)
			}
		})
	}
}

func TestADraftCreationWritesNoCanonical(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	deps.Bodies = &bodyLog{}
	deps.Types = typesStub{canonical: true}
	sc := publishContext(t)
	sc.Run.PublishMode = run.PublishDraft
	server.ResetRequests()

	runPublish(t, deps, sc)
	if sent := canonicalsSent(t, server); len(sent) != 0 {
		t.Fatalf("a draft creation sent the canonical %v, which WordPress gives a draft as ?page_id=", sent)
	}
}
