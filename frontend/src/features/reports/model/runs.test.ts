import { describe, expect, it } from "vitest";

import type { ItemReport, Run } from "../../../data/types.js";
import { finished, itemRows } from "./runs.js";

function run(id: string, status: string): Run {
    return {
        id,
        siteId: "site-1",
        kind: "generate",
        status,
        targets: null,
        recipe: null,
        templateId: "t-1",
        templateVersion: 1,
        publishMode: "draft",
        budget: { maxUsd: 0, maxTokens: 0 },
        stats: { items: 0, done: 0, failed: 0, tokens: 0, usd: 0 },
        createdBy: "user",
        parentRunId: null,
        pauseReason: "",
        error: "",
        deadlineAt: null,
        createdAt: null,
        startedAt: null,
        finishedAt: null,
    };
}

function item(overrides: Partial<ItemReport>): ItemReport {
    return { itemId: "i-1", pageId: "p-1", status: "completed", error: "", ...overrides };
}

describe("finished", () => {
    it("keeps only the runs that have settled", () => {
        const rows = [run("a", "running"), run("b", "completed"), run("c", "failed"), run("d", "paused"), run("e", "cancelled")];
        expect(finished(rows).map((held) => held.id)).toEqual(["b", "c", "e"]);
    });
});

describe("itemRows", () => {
    const paths = new Map([
        ["p-1", "/a/"],
        ["p-2", "/b/"],
    ]);

    it("reads the path, the score and the finding counts out of the final report", () => {
        const rows = itemRows(
            [item({ itemId: "i-1", pageId: "p-1", report: { path: "/a/", score: 8.5, errors: 0, warnings: 2 } })],
            paths,
        );
        expect(rows[0]).toMatchObject({ path: "/a/", score: 8.5, errors: 0, warnings: 2 });
    });

    it("falls back to the page map when no report was written", () => {
        const rows = itemRows([item({ itemId: "i-2", pageId: "p-2", status: "failed", error: "boom" })], paths);
        expect(rows[0]).toMatchObject({ path: "/b/", errors: 0, warnings: 0, score: null, error: "boom" });
    });

    it("puts the failed items first and the worst of the rest above the clean ones", () => {
        const rows = itemRows(
            [
                item({ itemId: "a", pageId: "p-1", status: "completed", report: { path: "/a/", errors: 0 } }),
                item({ itemId: "b", pageId: "p-2", status: "failed" }),
                item({ itemId: "c", pageId: "p-1", status: "completed", report: { path: "/c/", errors: 3 } }),
            ],
            paths,
        );
        expect(rows.map((held) => held.itemId)).toEqual(["b", "c", "a"]);
    });

    it("has nothing to show for a run with no items", () => {
        expect(itemRows(null, paths)).toEqual([]);
    });
});
