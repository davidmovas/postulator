import { describe, expect, it } from "vitest";

import type { PageTreeNode, Run } from "../../../data/types.js";
import { driftCount, edgeTile, entityTile, histogram, isEmptySite, pageTile, runSummary } from "./overview.js";

function run(id: string, status: string, done = 0, items = 0): Run {
    return {
        id,
        siteId: "s1",
        kind: "generate",
        status,
        targets: null,
        recipe: null,
        templateId: "t1",
        templateVersion: 1,
        publishMode: "draft",
        budget: { maxUsd: 25, maxTokens: 0 },
        stats: { items, done, failed: 0, tokens: 0, usd: 3.5 },
        createdBy: "schedule",
        parentRunId: null,
        pauseReason: "",
        error: "",
        deadlineAt: null,
        createdAt: "2026-09-20T10:00:00Z",
        startedAt: "2026-09-20T10:00:01Z",
        finishedAt: null,
    };
}

function node(id: string, drift: boolean, children: PageTreeNode[] = []): PageTreeNode {
    return {
        page: {
            id,
            siteId: "s1",
            path: `/${id}/`,
            slug: id,
            parentPageId: null,
            wpType: "page",
            wpId: 1,
            title: id,
            h1: id,
            metaTitle: "",
            metaDescription: "",
            canonical: "",
            status: "published",
            entityId: null,
            templateId: null,
            contentHash: "",
            observed: { link: "", slug: "", status: "", title: "", h1: "" },
            mismatches: [],
            wpModifiedAt: null,
            lastSyncedAt: null,
            drift,
            createdAt: null,
            updatedAt: null,
        },
        children,
    };
}

describe("the run summary", () => {
    it("prefers a run that is still going over a newer finished one", () => {
        const summary = runSummary([run("r2", "completed", 10, 10), run("r1", "running", 4, 12)]);
        expect(summary?.run.id).toBe("r1");
        expect(summary?.active).toBe(true);
        expect(summary?.done).toBe(4);
        expect(summary?.total).toBe(12);
        expect(summary?.maxUsd).toBe(25);
    });

    it("falls back to the newest run when nothing is going", () => {
        const summary = runSummary([run("r2", "completed", 10, 10), run("r1", "failed", 1, 3)]);
        expect(summary?.run.id).toBe("r2");
        expect(summary?.active).toBe(false);
    });

    it("answers nothing when the site has never run", () => {
        expect(runSummary([])).toBeNull();
    });
});

describe("the tiles", () => {
    it("counts entities without a canonical page from the difference", () => {
        expect(entityTile({ total: 41, withCanonicalPage: 30, withPublishedPage: 22 })).toStrictEqual({
            total: 41,
            withCanonical: 30,
            withPublished: 22,
            withoutPage: 11,
        });
    });

    it("counts mapped pages from the total and sorts the statuses by weight", () => {
        const tile = pageTile({
            total: 62,
            unmapped: 18,
            orphans: 4,
            byStatus: { published: 40, draft: 20, archived: 0, planned: 2 },
        });
        expect(tile.mapped).toBe(44);
        expect(tile.byStatus.map((held) => held.status)).toStrictEqual(["published", "draft", "planned"]);
    });

    it("survives a site whose page statuses are null", () => {
        expect(pageTile({ total: 0, unmapped: 0, orphans: 0, byStatus: null }).byStatus).toStrictEqual([]);
    });

    it("reads the edge gap and the coverage off the approved and realised totals", () => {
        const tile = edgeTile({ approved: 614, realized: 436 }, 68, true);
        expect(tile.gap).toBe(178);
        expect(tile.coverage).toBeCloseTo(0.71, 2);
        expect(tile.capped).toBe(true);
        expect(edgeTile({ approved: 0, realized: 0 }, 0, false).coverage).toBe(0);
    });
});

describe("the charts", () => {
    it("scales every depth bar against the tallest one and sorts by depth", () => {
        const bars = histogram([
            { depth: 3, pages: 50 },
            { depth: 1, pages: 100 },
            { depth: 2, pages: 25 },
        ]);
        expect(bars.map((bar) => bar.depth)).toStrictEqual([1, 2, 3]);
        expect(bars.map((bar) => bar.fraction)).toStrictEqual([1, 0.25, 0.5]);
    });

    it("gives every bar a zero fraction when nothing has pages", () => {
        expect(histogram([{ depth: 1, pages: 0 }])[0]?.fraction).toBe(0);
    });
});

describe("drift over the page tree", () => {
    it("counts every drifted page at any depth", () => {
        const roots = [node("a", true, [node("b", false), node("c", true, [node("d", true)])]), node("e", false)];
        expect(driftCount(roots)).toBe(3);
    });

    it("counts nothing on an empty tree", () => {
        expect(driftCount([])).toBe(0);
    });
});

describe("an empty site", () => {
    it("is one with neither pages nor entities", () => {
        const none = { total: 0, unmapped: 0, orphans: 0, byStatus: null };
        expect(isEmptySite(none, { total: 0, withCanonicalPage: 0, withPublishedPage: 0 })).toBe(true);
        expect(isEmptySite(none, { total: 3, withCanonicalPage: 0, withPublishedPage: 0 })).toBe(false);
    });
});
