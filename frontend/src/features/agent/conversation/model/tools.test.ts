import { describe, expect, it } from "vitest";

import { familyOf, resultSummary, verbOf } from "./tools.js";

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
            "a long answer, 40 KB",
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
