package support_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/application/support"
	"github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	siteID       = "site-1"
	secretMarker = "s3cret-app-password-Zq9"
	titleMarker  = "Client Title Words"
	bodyMarker   = "the client's own paragraph"
	argsMarker   = "args-the-client-typed"
	errorMarker  = "error text quoting the client"
	cookieMarker = "wordpress_logged_in_cookie"
	crumbMarker  = "Crumb Name Of The Client"
)

var stamp = time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)

type world struct{}

func (world) Get(context.Context, string) (site.Site, error) {
	return site.Site{
		ID: siteID, Name: "Shop", BaseURL: "https://shop.example.com", Username: secretMarker, SecretRef: cookieMarker,
		Plugin: site.PluginState{Installed: true, Version: "1.5.2", Capabilities: []string{"rendered"}},
	}, nil
}

func (world) ListBySite(context.Context, string) ([]pagemap.Page, error) {
	hub := pagemap.Page{ID: "hub", SiteID: siteID, Path: "/serums/", WPType: pagemap.WPPage, WPID: new(int64(2)), Status: pagemap.StatusPublished,
		Title: titleMarker, EntityID: new("e-serums"), ParentReason: pagemap.ParentNone, PageTypeID: new("type-hub"), TypeReason: pagemap.TypeByRule}
	child := pagemap.Page{ID: "vitc", SiteID: siteID, Path: "/vitamin-c/", WPType: pagemap.WPPage, WPID: new(int64(3)), Status: pagemap.StatusPublished,
		Title: titleMarker, H1: bodyMarker, ParentPageID: new("hub"), ParentReason: pagemap.ParentJSONLD, ParentMethod: pagemap.MethodHref}
	return []pagemap.Page{hub, child}, nil
}

type renders struct{}

func (renders) ListBySite(context.Context, string) ([]render.Record, error) {
	return []render.Record{{PageID: "vitc", ExtractedAt: stamp, Extraction: render.Extraction{
		Root: render.RootFound,
		Trails: []pagemap.Trail{{Source: pagemap.TrailJSONLD, Crumbs: []pagemap.Crumb{
			{Name: crumbMarker, Path: "/serums/", Href: "https://shop.example.com/serums/"},
		}}},
	}}}, nil
}

type types struct{}

func (types) Set(context.Context, string) (pagetype.Set, error) {
	return pagetype.NewSet([]pagetype.Type{{ID: "type-hub", SiteID: siteID, Name: "Category", Preset: pagetype.Category}}, nil), nil
}

type entities struct{}

func (entities) ListBySite(context.Context, string) ([]graph.Entity, error) {
	return []graph.Entity{
		{ID: "e-serums", SiteID: siteID, Name: titleMarker, CanonicalPageID: new("hub"), Definition: bodyMarker},
		{ID: "e-loose", SiteID: siteID, Name: titleMarker},
	}, nil
}

type edges struct{}

func (edges) ListBySite(context.Context, string) ([]graph.Edge, error) {
	return []graph.Edge{{ID: "edge", SiteID: siteID, FromEntityID: "e-loose", ToEntityID: "e-serums", Kind: graph.EdgeRelated,
		Status: graph.StatusApproved, Reason: bodyMarker}}, nil
}

type findings struct{}

func (findings) ListBySite(context.Context, string, finding.Status) ([]finding.Finding, error) {
	return []finding.Finding{
		{ID: "f1", SiteID: siteID, Code: finding.EntityParentDiffers, Status: finding.Open, Subject: json.RawMessage(`{"entity":"` + titleMarker + `"}`)},
		{ID: "f2", SiteID: siteID, Code: finding.EntityParentDiffers, Status: finding.Open},
		{ID: "f3", SiteID: siteID, Code: finding.RenderUnreachable, Status: finding.Open},
	}, nil
}

type observation struct{}

func (observation) Summary(context.Context, string) (observe.Summary, error) {
	return observe.Summary{Pages: 2, Observed: 2, Rendered: 1, ByType: map[string]int{"page": 2}}, nil
}

type conversations struct{}

func (conversations) List(context.Context, agent.ConversationQuery, paging.Request) (paging.List[agent.Conversation], error) {
	return paging.List[agent.Conversation]{Items: []agent.Conversation{{ID: "conv-1", SiteID: new(siteID), Title: titleMarker, Mode: agent.ModeConfirm}}}, nil
}

type calls struct{}

func (calls) ByConversation(context.Context, string) ([]agent.ToolCall, error) {
	return []agent.ToolCall{
		{ID: "t1", ConversationID: "conv-1", CallID: "call-1", Tool: "pages_tree", Args: json.RawMessage(`{"q":"` + argsMarker + `"}`), Status: agent.CallOK, DurationMS: 12},
		{ID: "t2", ConversationID: "conv-1", CallID: "call-2", Tool: "pages_get", Status: agent.CallError, Error: errorMarker, DurationMS: 7},
	}, nil
}

type messages struct{}

func (messages) List(context.Context, agent.MessageQuery, paging.Request) (paging.List[agent.Message], error) {
	return paging.List[agent.Message]{Items: []agent.Message{
		{ID: "m1", ConversationID: "conv-1", Role: agent.RoleUser, Text: bodyMarker},
		{ID: "m2", ConversationID: "conv-1", Role: agent.RoleTool, CallID: "call-1", Payload: json.RawMessage(`{"nodes":"` + strings.Repeat("x", 300) + `"}`)},
	}}, nil
}

func service() *support.Service {
	return support.New(support.Deps{
		Version: "2.7.0", PluginRequired: "1.5.1", Sites: world{}, Pages: world{}, Renders: renders{}, Types: types{},
		Entities: entities{}, Edges: edges{}, Findings: findings{}, Observation: observation{}, Conversations: conversations{},
		Calls: calls{}, Messages: messages{}, Clock: clock.NewFake(stamp),
	})
}

func TestASupportBundleSaysWhatTheModelHoldsAndNothingTheClientWrote(t *testing.T) {
	t.Parallel()

	bundle, err := service().Bundle(t.Context(), support.BundleRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("Bundle: %v", err)
	}
	encoded, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	text := string(encoded)
	for _, never := range []string{secretMarker, titleMarker, bodyMarker, argsMarker, errorMarker, cookieMarker, crumbMarker} {
		if strings.Contains(text, never) {
			t.Errorf("the bundle carries %q:\n%s", never, text)
		}
	}
	switch {
	case bundle.App.Version != "2.7.0" || bundle.Plugin.Version != "1.5.2" || bundle.Plugin.Required != "1.5.1":
		t.Errorf("the versions = %+v %+v", bundle.App, bundle.Plugin)
	case bundle.Observation.Pages != 2 || len(bundle.Pages) != 2:
		t.Errorf("the observation and pages = %+v, %d", bundle.Observation, len(bundle.Pages))
	case bundle.Graph.Entities != 2 || bundle.Graph.WithoutParent != 2 || bundle.Graph.WithoutPage != 1 || bundle.Graph.RelatedEdges != 1:
		t.Errorf("the graph's shape = %+v", bundle.Graph)
	case bundle.Findings["entity_parent_differs"] != 2 || bundle.Findings["render_unreachable"] != 1:
		t.Errorf("the findings by code = %v", bundle.Findings)
	case len(bundle.Calls) != 2 || bundle.Calls[0].Tool != "pages_tree" || bundle.Calls[0].ResultBytes == 0 || bundle.Calls[1].Status != "error":
		t.Errorf("the calls = %+v", bundle.Calls)
	}
	byPath := map[string]support.PageFacts{}
	for _, page := range bundle.Pages {
		byPath[page.Path] = page
	}
	if hub := byPath["/serums/"]; hub.Type != "Category" || hub.ParentReason != "none" || hub.Why == "" || !hub.Entity {
		t.Errorf("the hub = %+v", hub)
	}
	child := byPath["/vitamin-c/"]
	if child.Parent != "/serums/" || child.ParentMethod != "href" || child.Render == nil || child.Render.Root != "found" ||
		len(child.Render.Crumbs) != 1 || child.Render.Crumbs[0][0].Path != "/serums/" {
		t.Errorf("the child = %+v", child)
	}
}

func TestASupportBundleIsWrittenToTheFileThePersonChose(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "support.json")
	written, err := service().Write(t.Context(), support.WriteRequest{SiteID: siteID, Path: path})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	held, err := os.ReadFile(path)
	if err != nil || int64(len(held)) != written.Bytes || !json.Valid(held) {
		t.Fatalf("the file holds %d bytes (%v), the write said %d", len(held), err, written.Bytes)
	}
	if _, err = service().Write(t.Context(), support.WriteRequest{SiteID: siteID}); err == nil {
		t.Error("a bundle with nowhere to go was written")
	}
}
