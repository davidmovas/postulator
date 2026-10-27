import { describe, expect, it } from "vitest";

import { copy } from "../../../../copy/index.js";
import { actOf, familyOf, refusedText, resultSummary, toolLabel, truncationOf } from "./tools.js";

const folded = [
    "sites",
    "sites_manage",
    "pages_find",
    "pages_tree",
    "pages_get",
    "pages_manage",
    "pages_fields",
    "pages_blocks",
    "pages_judge",
    "links",
    "graph_read",
    "graph_edit",
    "graph_propose",
    "pagetypes",
    "pagetypes_classify",
    "templates",
    "runs",
    "runs_control",
    "imports",
    "findings",
    "models",
    "schedules",
    "plans",
];

describe("familyOf", () => {
    it("files every tool of the fold under a family of its own", () => {
        for (const tool of folded) {
            expect(familyOf(tool), tool).not.toBe("other");
        }
        expect(familyOf("links")).toBe("links");
        expect(familyOf("pagetypes_classify")).toBe("pagetypes");
        expect(familyOf("plans")).toBe("plans");
    });

    it("keeps an unknown tool readable", () => {
        expect(familyOf("mystery")).toBe("other");
        expect(familyOf("")).toBe("other");
    });
});

describe("actOf", () => {
    it("reads the op of a write and the view of a read", () => {
        expect(actOf("graph_edit", { op: "move", entityId: "e1" })).toBe("move");
        expect(actOf("pages_tree", { view: "walk" })).toBe("walk");
    });

    it("falls back to the tool's own default", () => {
        expect(actOf("findings", {})).toBe("list");
        expect(actOf("pages_get", { pageId: "p1" })).toBe("model");
        expect(actOf("graph_edit", null)).toBe("");
    });
});

describe("toolLabel", () => {
    it.each([
        ["pages_find", {}, "Find pages"],
        ["pages_tree", { view: "walk" }, "Walk the tree"],
        ["pages_tree", { view: "subtree", pageId: "p1" }, "Read a branch"],
        ["graph_read", { view: "entities" }, "List entities"],
        ["runs_control", { op: "start" }, "Start a run"],
        ["sites_manage", { op: "sync" }, "Sync the site"],
        ["pagetypes", { op: "get", pageTypeId: "pt1" }, "Read a page type"],
        ["findings", {}, "List findings"],
    ])("names %s %j by what it does", (tool, args, label) => {
        expect(toolLabel(tool, args)).toBe(label);
    });

    it("names a tool by itself when its op is not known", () => {
        expect(toolLabel("graph_read")).toBe("Read the graph");
        expect(toolLabel("graph_edit", { op: "teleport" })).toBe("Change the graph");
    });

    it("has words for every tool of the fold and carries no underscore", () => {
        for (const tool of folded) {
            expect(copy.agent.acts[tool], tool).toBeDefined();
            expect(toolLabel(tool)).not.toContain("_");
        }
        expect(toolLabel("mystery_move")).toBe("Mystery move");
    });
});

describe("resultSummary", () => {
    it("names a confirmation request", () => {
        expect(resultSummary("pages_manage", { status: "confirmationRequired", actionId: "a1", summary: "x" })).toBe(
            "asked for approval",
        );
    });

    it("counts a list, bare or wrapped by the view that answered it", () => {
        expect(resultSummary("pages_find", { items: [{}, {}, {}], hasMore: true })).toBe("3 pages, more available");
        expect(resultSummary("graph_read", { entities: { items: [], hasMore: false } })).toBe("no entities");
        expect(resultSummary("graph_read", { edges: { items: [{}], hasMore: false } })).toBe("1 edge");
        expect(resultSummary("sites", { sites: { items: [{}], hasMore: false } })).toBe("1 site");
        expect(resultSummary("runs", { items: { items: [{}, {}], hasMore: false } })).toBe("2 pages");
    });

    it("names the record a read or a write answered with", () => {
        expect(resultSummary("graph_read", { entity: { id: "e1", name: "Ceramic Mugs" } })).toBe("Ceramic Mugs");
        expect(resultSummary("pages_manage", { page: { id: "p1", path: "/mugs/" } })).toBe("/mugs/");
        expect(resultSummary("templates", { template: { id: "t1", name: "Guide" } })).toBe("Guide");
        expect(resultSummary("runs_control", { runId: "r1", estimate: {} })).toBe("run started");
    });

    it("says what a truncated or empty answer is", () => {
        expect(resultSummary("pages_tree", { truncated: true, totalBytes: 40_000, preview: "…" })).toBe("40 KB, shortened to fit");
        expect(resultSummary("pages_manage", {})).toBe("done");
        expect(resultSummary("pages_manage", null)).toBe("done");
        expect(resultSummary("runs_control", { cancelled: true })).toBe("done");
    });

    it("says which part of the site's model the agent read", () => {
        const said = copy.agent.transcript.tool.explore;
        const page = { pageId: "p1", path: "/peptides/", title: "Peptides" };
        expect(resultSummary("pages_get", { page: { page, parent: { trail: [] } }, template: null })).toBe(said.page("/peptides/"));
        expect(resultSummary("pages_get", { pack: { path: "/peptides/", tokens: 1840, text: "…" } })).toBe(
            said.pack("/peptides/", "1,840"),
        );
        expect(resultSummary("pages_tree", { node: { page }, nodes: { items: [{}, {}], hasMore: true }, generation: 3 })).toBe(
            "2 pages under /peptides/, more available",
        );
        expect(resultSummary("pages_tree", { nodes: { items: [{}], hasMore: false }, generation: 3 })).toBe(
            "1 page at the top of the site",
        );
        expect(resultSummary("links", { view: "page", page: { path: "/peptides/" } })).toBe(said.links("/peptides/"));
        expect(resultSummary("links", { view: "owed", owed: { items: [{}, {}, {}], hasMore: false } })).toBe(said.owed(3, false));
        expect(resultSummary("links", { view: "section", root: { path: "/kits/" }, totals: {} })).toBe(said.section("/kits/"));
        expect(resultSummary("graph_read", { adoption: { adopting: 2, skipped: 0, steps: { items: [] } } })).toBe(
            copy.agent.transcript.tool.adoption(2),
        );
    });

    it("names a site's overview and a run read whole", () => {
        const said = copy.agent.transcript.tool;
        expect(resultSummary("sites", { overview: { site: {}, totals: {}, brief: {} } })).toBe(said.overview);
        expect(resultSummary("runs", { run: { id: "r1", kind: "generate", status: "running" } })).toBe(
            said.runs.run("generate", "running"),
        );
        expect(resultSummary("runs", { events: { events: [{}, {}] } })).toBe(said.runs.events(2));
        expect(resultSummary("findings", { list: { items: [{}], hasMore: false } })).toBe("1 finding");
        expect(resultSummary("pagetypes", { types: [{}, {}], fields: [] })).toBe("2 page types");
    });

    it("tells proposed cards from written ones", () => {
        expect(resultSummary("graph_propose", { proposals: [{}, {}], skipped: [] })).toBe("proposed 2 cards for review");
        expect(resultSummary("graph_propose", { entities: [{}], skipped: [{}] })).toBe("wrote 1 card, 1 left as it was");
    });

    it("never leaks JSON into the summary", () => {
        const summary = resultSummary("mystery", { site: {}, totals: {} });
        expect(summary).not.toMatch(/[{}"]/);
        expect(summary).toBe("2 fields");
    });
});

describe("truncationOf", () => {
    it("reads what a shortened answer says it dropped, widest list first", () => {
        const cut = truncationOf({
            truncated: true,
            totalBytes: 41_000,
            droppedItems: { "tree.children": 4, items: 37 },
            shortenedText: 2,
            result: { items: [{ id: "p1" }], nextCursor: "c1" },
        });
        expect(cut).not.toBeNull();
        expect(cut?.totalBytes).toBe(41_000);
        expect(cut?.shortened).toBe(2);
        expect(cut?.whole).toBe(true);
        expect(cut?.dropped).toStrictEqual([
            { path: "items", count: 37 },
            { path: "tree.children", count: 4 },
        ]);
    });

    it("marks a fallback preview as no longer whole", () => {
        const cut = truncationOf({ truncated: true, totalBytes: 41_000, preview: '{"items":[' });
        expect(cut?.whole).toBe(false);
        expect(cut?.dropped).toStrictEqual([]);
    });

    it("answers nothing for an answer that was never cut", () => {
        expect(truncationOf({ items: [] })).toBeNull();
        expect(truncationOf(null)).toBeNull();
        expect(truncationOf("done")).toBeNull();
    });
});

describe("refusedText", () => {
    it("knows the sentence the guard writes when a tool is closed to a conversation", () => {
        expect(refusedText("the tool pages_manage is not open to this conversation")).toBe(true);
        expect(refusedText("no record carries that id")).toBe(false);
        expect(refusedText("")).toBe(false);
    });
});
