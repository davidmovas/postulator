import { describe, expect, it } from "vitest";

import type { Entity, Page, PageAudit, PageTreeNode } from "../../../data/types.js";
import { auditCard, averageDepth, bars, coverageRows, driftRows, flattenTree, noPage, tiles } from "./site.js";

function page(overrides: Partial<Page> = {}): Page {
    return {
        id: "p-1",
        siteId: "site-1",
        path: "/a/",
        slug: "a",
        parentPageId: null,
        wpType: "page",
        wpId: 1,
        title: "A",
        h1: "A",
        metaTitle: "",
        metaDescription: "",
        canonical: "",
        primaryKeyword: "",
        keywords: [],
        status: "published",
        entityId: "e-1",
        templateId: null,
        contentHash: "",
        observed: { link: "", slug: "", status: "", title: "", h1: "" },
        mismatches: [],
        wpModifiedAt: null,
        lastSyncedAt: null,
        drift: false,
        createdAt: null,
        updatedAt: null,
        ...overrides,
    };
}

function entity(overrides: Partial<Entity> = {}): Entity {
    return {
        id: "e-1",
        siteId: "site-1",
        name: "Alpha",
        kind: "topic",
        intent: "",
        primaryKeyword: "",
        secondaryKeywords: null,
        anchors: null,
        canonicalPageId: "p-1",
        score: 0,
        source: "user",
        createdAt: null,
        updatedAt: null,
        ...overrides,
    };
}

function audit(overrides: Partial<PageAudit> = {}): PageAudit {
    return {
        pageId: "p-1",
        path: "/a/",
        status: "published",
        entityId: "e-1",
        entityName: "Alpha",
        skipReason: "",
        onSite: true,
        targets: 3,
        required: 2,
        satisfied: 2,
        missing: 0,
        missingRequired: 0,
        blocked: 0,
        offGraph: 0,
        pending: 0,
        unpublished: 0,
        inbound: 1,
        orphan: false,
        ...overrides,
    };
}

describe("tiles", () => {
    it("measures what the overview really carries", () => {
        const held = tiles(
            { total: 10, withCanonicalPage: 7, withPublishedPage: 4 },
            { total: 20, unmapped: 5, orphans: 3, byStatus: null },
            [
                { depth: 0, pages: 2 },
                { depth: 1, pages: 8 },
            ],
        );
        expect(held.pagesMapped).toBe(15);
        expect(held.mappedShare).toBe(0.75);
        expect(held.withPageShare).toBe(0.7);
        expect(held.orphans).toBe(3);
        expect(held.averageDepth).toBeCloseTo(1.8, 5);
    });

    it("divides by nothing without dividing by zero", () => {
        const held = tiles({ total: 0, withCanonicalPage: 0, withPublishedPage: 0 }, { total: 0, unmapped: 0, orphans: 0, byStatus: null }, null);
        expect(held.mappedShare).toBe(0);
        expect(held.withPageShare).toBe(0);
        expect(held.averageDepth).toBe(0);
    });
});

describe("averageDepth counts the root as depth one", () => {
    it.each([
        [[{ depth: 0, pages: 1 }], 1],
        [
            [
                { depth: 0, pages: 1 },
                { depth: 1, pages: 1 },
            ],
            1.5,
        ],
        [[], 0],
    ])("reads %j as %s", (buckets, want) => {
        expect(averageDepth(buckets)).toBeCloseTo(want, 5);
    });
});

describe("bars", () => {
    it("scales every bar against the tallest and numbers the levels from one", () => {
        expect(
            bars([
                { depth: 1, pages: 10 },
                { depth: 0, pages: 5 },
            ]),
        ).toEqual([
            { level: 1, pages: 5, fraction: 0.5 },
            { level: 2, pages: 10, fraction: 1 },
        ]);
    });

    it("has nothing to draw for no page", () => {
        expect(bars(null)).toEqual([]);
    });
});

describe("the page tree", () => {
    const roots: PageTreeNode[] = [
        {
            page: page({ id: "p-1", path: "/a/" }),
            children: [{ page: page({ id: "p-2", path: "/a/b/", drift: true, wpModifiedAt: "2026-09-18T00:00:00Z" }), children: null }],
        },
        { page: page({ id: "p-3", path: "/c/", drift: true, wpModifiedAt: "2026-09-20T00:00:00Z" }), children: null },
    ];

    it("flattens depth first", () => {
        expect(flattenTree(roots).map((held) => held.id)).toEqual(["p-1", "p-2", "p-3"]);
        expect(flattenTree(null)).toEqual([]);
    });

    it("keeps the drifted pages, newest change first", () => {
        expect(driftRows(flattenTree(roots)).map((held) => held.id)).toEqual(["p-3", "p-2"]);
    });
});

describe("coverageRows", () => {
    const pages = [
        page({ id: "p-1", path: "/a/", status: "published" }),
        page({ id: "p-2", path: "/b/", status: "planned" }),
    ];
    const entities = [
        entity({ id: "e-1", name: "Alpha", canonicalPageId: "p-1" }),
        entity({ id: "e-2", name: "Beta", canonicalPageId: "p-2" }),
        entity({ id: "e-3", name: "Gamma", canonicalPageId: null }),
    ];
    const audits = [audit({ pageId: "p-1", satisfied: 4 }), audit({ pageId: "p-2", satisfied: 1 })];

    it("puts the entities with no page first and the published last", () => {
        expect(coverageRows(entities, pages, audits).map((row) => row.name)).toEqual(["Gamma", "Beta", "Alpha"]);
    });

    it("carries the page state and the links that are really in place", () => {
        const rows = coverageRows(entities, pages, audits);
        expect(rows[0]).toMatchObject({ reason: noPage, links: 0, path: "" });
        expect(rows[1]).toMatchObject({ reason: "planned", links: 1, path: "/b/" });
        expect(rows[2]).toMatchObject({ reason: "published", links: 4 });
    });

    it("breaks a tie on the weaker linking", () => {
        const two = [
            entity({ id: "e-1", name: "Alpha", canonicalPageId: "p-1" }),
            entity({ id: "e-4", name: "Delta", canonicalPageId: "p-4" }),
        ];
        const both = [page({ id: "p-1", status: "published" }), page({ id: "p-4", path: "/d/", status: "published" })];
        const rows = coverageRows(two, both, [audit({ pageId: "p-1", satisfied: 4 })]);
        expect(rows.map((row) => row.name)).toEqual(["Delta", "Alpha"]);
    });
});

describe("auditCard", () => {
    it("counts a compliant page as one that misses nothing and is not blocked", () => {
        const held = auditCard(
            [
                audit({ pageId: "p-1" }),
                audit({ pageId: "p-2", missing: 2, missingRequired: 1 }),
                audit({ pageId: "p-3", skipReason: "unmapped" }),
            ],
            {
                pages: 5,
                audited: 2,
                required: 6,
                satisfied: 4,
                missing: 2,
                missingRequired: 1,
                blocked: 0,
                offGraph: 3,
                orphans: 1,
                pending: 0,
                unpublished: 0,
            },
        );
        expect(held.compliant).toBe(1);
        expect(held.compliantShare).toBe(0.5);
        expect(held.audited).toBe(2);
        expect(held.offGraph).toBe(3);
    });

    it("measures compliance over the pages on the site and counts what waits apart", () => {
        const held = auditCard(
            [
                audit({ pageId: "p-1" }),
                audit({ pageId: "p-2", status: "planned", onSite: false, satisfied: 0, pending: 3 }),
                audit({ pageId: "p-3", missing: 1 }),
            ],
            {
                pages: 3,
                audited: 3,
                required: 6,
                satisfied: 2,
                missing: 1,
                missingRequired: 0,
                blocked: 0,
                offGraph: 0,
                orphans: 0,
                pending: 3,
                unpublished: 0,
            },
        );
        expect(held.compliant).toBe(1);
        expect(held.compliantShare).toBe(0.5);
        expect(held.pending).toBe(3);
    });
});
