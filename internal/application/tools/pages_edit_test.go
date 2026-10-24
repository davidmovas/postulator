package tools_test

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	clientOpening = "<p>The owner's own opening about coffee.</p>"
	ourStorage    = "<p>Our paragraph on keeping beans fresh.</p>"
	coffeeBody    = clientOpening + "\n\n" + ourStorage
)

func editable(t *testing.T, seeded fixture) {
	t.Helper()

	onTheSite(t, seeded)
	ctx := t.Context()
	if err := sqlite.NewPageFieldRepo(seeded.store).Observe(ctx, seeded.page, []field.Value{
		field.NewValue(seeded.page, field.Excerpt, "The owner's excerpt.", stamp),
	}); err != nil {
		t.Fatalf("observe the fields: %v", err)
	}
	bodies := sqlite.NewPageBodyRepo(seeded.store)
	if err := bodies.Observe(ctx, block.Snapshot{
		PageID: seeded.page, Side: block.Current, Raw: coffeeBody, Hash: block.BodyHash(coffeeBody), Kind: block.Classic, ObservedAt: stamp,
	}); err != nil {
		t.Fatalf("observe the body: %v", err)
	}
	if err := bodies.ReplaceWritten(ctx, seeded.page, []string{block.Parse(coffeeBody).Blocks[1].Hash}, "run-1", stamp); err != nil {
		t.Fatalf("record our block: %v", err)
	}
	if err := sqlite.NewSiteRepo(seeded.store).UpdatePlugin(ctx, seeded.site, site.PluginState{Installed: true}, "", stamp); err != nil {
		t.Fatalf("install the plugin: %v", err)
	}
}

func callTool(t *testing.T, registry *tools.Registry, binding tools.Binding, name, args string) (any, error) {
	t.Helper()
	return registry.Call(t.Context(), binding, name, json.RawMessage(args))
}

func TestPagesFieldsReadsAndEveryWriteWaitsForAPerson(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	editable(t, seeded)
	binding.Mode = domainagent.ModeConfirm

	out, err := callTool(t, registry, binding, "pages_fields", `{"op":"get","pageId":"`+seeded.page+`"}`)
	read, ok := out.(tools.PagesFieldsResponse)
	if err != nil || !ok || read.Fields == nil || len(read.Fields.Fields.Items) != 1 || read.Fields.Fields.Items[0].Hash != field.Hash("The owner's excerpt.") {
		t.Fatalf("get = %+v, %v, want the excerpt with its hash and no card", out, err)
	}

	set := `{"op":"set","pageId":"` + seeded.page + `","key":"excerpt","value":"Pull a balanced shot.","hash":"` + field.Hash("The owner's excerpt.") + `"}`
	for _, mode := range []domainagent.Mode{domainagent.ModeConfirm, domainagent.ModeAutonomous} {
		binding.Mode = mode
		out, err = callTool(t, registry, binding, "pages_fields", set)
		asked, waited := out.(tools.Confirmation)
		if err != nil || !waited || !strings.Contains(asked.Summary, "/coffee/") || !strings.Contains(asked.Summary, "Pull a balanced shot.") {
			t.Fatalf("set in %s mode = %+v, %v, want a card naming the page and the new value", mode, out, err)
		}
	}
	binding.Approved = true
	out, err = callTool(t, registry, binding, "pages_fields", set)
	if started, ok := out.(tools.PagesFieldsResponse); err != nil || !ok || started.Edit == nil || started.Edit.RunID == "" {
		t.Fatalf("the approved set = %+v, %v, want the edit run it started", out, err)
	}

	binding.Approved = false
	refusals := []struct {
		name string
		args string
		code errors.Code
	}{
		{name: "a denied field", args: `{"op":"set","pageId":"` + seeded.page + `","key":"sku","value":"X-1"}`, code: errors.Invalid},
		{name: "a read-only field", args: `{"op":"set","pageId":"` + seeded.page + `","key":"h1","value":"x"}`, code: errors.Invalid},
		{name: "a stale hash", args: `{"op":"set","pageId":"` + seeded.page + `","key":"excerpt","value":"x","hash":"` + field.Hash("older") + `"}`, code: errors.Conflict},
	}
	for _, tc := range refusals {
		if _, err = callTool(t, registry, binding, "pages_fields", tc.args); !errors.IsCode(err, tc.code) {
			t.Errorf("%s = %v, want %s before any card", tc.name, err, tc.code)
		}
	}
}

func TestPagesBlocksListsAndRefusesTheOwnersBlockWithoutRewrites(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	editable(t, seeded)
	binding.Mode = domainagent.ModeAutonomous
	hash := block.BodyHash(coffeeBody)

	out, err := callTool(t, registry, binding, "pages_blocks", `{"op":"list","path":"/coffee/"}`)
	listed, ok := out.(tools.PagesBlocksResponse)
	if err != nil || !ok || listed.Blocks == nil || listed.Blocks.BodyHash != hash || len(listed.Blocks.Blocks.Items) != 2 ||
		listed.Blocks.Blocks.Items[0].Owner != sitemodel.OwnerClient || listed.Blocks.Blocks.Items[1].Owner != sitemodel.OwnerOurs {
		t.Fatalf("list = %+v, %v, want the owner's block then ours, with the body's hash", out, err)
	}

	cases := []struct {
		name    string
		args    string
		code    errors.Code
		summary string
	}{
		{name: "our block replaced", args: `{"op":"replace","pageId":"` + seeded.page + `","index":1,"html":"<p>Fresher.</p>","hash":"` + hash + `"}`,
			summary: "Replace block 1 of /coffee/"},
		{name: "a block inserted at the start", args: `{"op":"insert_after","pageId":"` + seeded.page + `","index":-1,"html":"<h2>Roasts</h2>","hash":"` + hash + `"}`,
			summary: "the start of /coffee/"},
		{name: "a link in the owner's block", args: `{"op":"place_link","pageId":"` + seeded.page + `","index":0,"targetId":"` + seeded.page + `","anchor":"coffee","hash":"` + hash + `"}`,
			summary: "link"},
		{name: "the owner's block replaced", args: `{"op":"replace","pageId":"` + seeded.page + `","index":0,"html":"<p>Ours.</p>","hash":"` + hash + `"}`,
			code: errors.Invalid},
		{name: "the owner's block deleted", args: `{"op":"delete","pageId":"` + seeded.page + `","index":0,"hash":"` + hash + `"}`, code: errors.Invalid},
		{name: "a stale hash", args: `{"op":"delete","pageId":"` + seeded.page + `","index":1,"hash":"` + block.BodyHash("older") + `"}`, code: errors.Conflict},
		{name: "a block past the end", args: `{"op":"delete","pageId":"` + seeded.page + `","index":` + strconv.Itoa(7) + `,"hash":"` + hash + `"}`, code: errors.Invalid},
		{name: "no index", args: `{"op":"delete","pageId":"` + seeded.page + `","hash":"` + hash + `"}`, code: errors.Invalid},
	}
	for _, tc := range cases {
		out, err = callTool(t, registry, binding, "pages_blocks", tc.args)
		if tc.code != "" {
			if !errors.IsCode(err, tc.code) {
				t.Errorf("%s = %+v, %v, want %s before any card", tc.name, out, err, tc.code)
			}
			continue
		}
		asked, waited := out.(tools.Confirmation)
		if err != nil || !waited || !strings.Contains(asked.Summary, tc.summary) {
			t.Errorf("%s = %+v, %v, want a card that says %q", tc.name, out, err, tc.summary)
		}
	}

	binding.Approved = true
	out, err = callTool(t, registry, binding, "pages_blocks", cases[0].args)
	if started, ok := out.(tools.PagesBlocksResponse); err != nil || !ok || started.Edit == nil || started.Edit.RunID == "" {
		t.Fatalf("the approved replace = %+v, %v, want the edit run it started", out, err)
	}
}

func TestPagesBlocksWaitsForThePlugin(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	editable(t, seeded)
	if err := sqlite.NewSiteRepo(seeded.store).UpdatePlugin(t.Context(), seeded.site, site.PluginState{}, "", stamp); err != nil {
		t.Fatalf("remove the plugin: %v", err)
	}
	binding.Mode = domainagent.ModeAutonomous
	_, err := callTool(t, registry, binding, "pages_blocks",
		`{"op":"delete","pageId":"`+seeded.page+`","index":1,"hash":"`+block.BodyHash(coffeeBody)+`"}`)
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("a block edit without the plugin = %v, want it refused", err)
	}
}
