import { describe, expect, it } from "vitest";

import { familyOf, refusedText, resultSummary, toolLabel, truncationOf, verbOf } from "./tools.js";

describe("familyOf and verbOf", () => {
    it("reads the family off the prefix and the verb off the rest", () => {
        expect(familyOf("graph_list_entities")).toBe("graph");
        expect(verbOf("graph_list_entities")).toBe("list entities");
        expect(familyOf("policies_effective")).toBe("policies");
        expect(verbOf("policies_effective")).toBe("effective");
        expect(familyOf("content_judge_page")).toBe("content");
        expect(verbOf("content_judge_page")).toBe("judge page");
    });

    it("keeps an unknown tool readable", () => {
        expect(familyOf("mystery")).toBe("other");
        expect(verbOf("mystery")).toBe("mystery");
        expect(familyOf("")).toBe("other");
    });
});

describe("resultSummary", () => {
    it("names a confirmation request", () => {
        expect(resultSummary("pages_delete", { status: "confirmationRequired", actionId: "a1", summary: "x" })).toBe(
            "asked for approval",
        );
    });

    it("counts a list", () => {
        expect(resultSummary("pages_list", { items: [{}, {}, {}], hasMore: true })).toBe("3 pages, more available");
        expect(resultSummary("graph_list_entities", { items: [], hasMore: false })).toBe("no entities");
        expect(resultSummary("sites_list", { items: [{}], hasMore: false })).toBe("1 site");
    });

    it("names the record a read or a write answered with", () => {
        expect(resultSummary("graph_get_entity", { entity: { id: "e1", name: "Ceramic Mugs" } })).toBe("Ceramic Mugs");
        expect(resultSummary("pages_update", { page: { id: "p1", path: "/mugs/" } })).toBe("/mugs/");
        expect(resultSummary("templates_create", { template: { id: "t1", name: "Guide" } })).toBe("Guide");
        expect(resultSummary("runs_start", { runId: "r1", estimate: {} })).toBe("run started");
    });

    it("says what a truncated or empty answer is", () => {
        expect(resultSummary("pages_tree", { truncated: true, totalBytes: 40_000, preview: "…" })).toBe(
            "40 KB, shortened to fit",
        );
        expect(resultSummary("pages_delete", {})).toBe("done");
        expect(resultSummary("pages_delete", null)).toBe("done");
        expect(resultSummary("runs_cancel", { cancelled: true })).toBe("done");
    });

    it("never leaks JSON into the summary", () => {
        const summary = resultSummary("reports_site_overview", { entities: { total: 3 }, pages: { total: 9, byStatus: {} } });
        expect(summary).not.toMatch(/[{}"]/);
        expect(summary).toBe("2 fields");
    });
});

describe("toolLabel", () => {
    it("names the act in words, never the wire name", () => {
        expect(toolLabel("pages_update")).toBe("Update page");
        expect(toolLabel("pages_list")).toBe("List pages");
        expect(toolLabel("graph_create_entity")).toBe("Create entity");
        expect(toolLabel("sites_get")).toBe("Get site");
        expect(toolLabel("schedules_search")).toBe("Search schedules");
    });

    it("spells out a tool of no known family and carries no underscore", () => {
        expect(toolLabel("mystery_move")).toBe("Mystery move");
        expect(toolLabel("mystery")).toBe("Mystery");
        for (const name of ["pages_update", "graph_create_entity", "mystery_move"]) {
            expect(toolLabel(name)).not.toContain("_");
        }
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
        const cut = truncationOf({ truncated: true, totalBytes: 41_000, preview: "{\"items\":[" });
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
        expect(refusedText("the tool pages_delete is not open to this conversation")).toBe(true);
        expect(refusedText("no record carries that id")).toBe(false);
        expect(refusedText("")).toBe(false);
    });
});
