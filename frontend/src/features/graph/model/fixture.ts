import type { Edge, Entity, EntityPage, PageType } from "../../../data/types.js";

export interface EntitySeed {
    id: string;
    name: string;
    score?: number;
    page?: boolean;
    source?: string;
    primaryKeyword?: string;
    secondaryKeywords?: string[];
    scope?: string;
}

export function entity(seed: EntitySeed): Entity {
    return {
        id: seed.id,
        siteId: "site",
        name: seed.name,
        intent: "",
        definition: "",
        facts: [],
        notes: [],
        keywords: [seed.primaryKeyword ?? seed.name.toLowerCase(), ...(seed.secondaryKeywords ?? [])].map((text) => ({ text })),
        anchors: [{ text: seed.name, source: "user", weight: 1 }],
        scopeEntityId: seed.scope ?? null,
        canonicalPageId: seed.page === false ? null : `page-${seed.id}`,
        score: seed.score ?? 0.5,
        source: seed.source ?? "user",
        createdAt: "2026-09-19T10:00:00Z",
        updatedAt: "2026-09-19T10:00:00Z",
    };
}

export function edge(id: string, from: string, to: string, kind: string, status = "approved", weight = 1, reason = ""): Edge {
    return {
        id,
        siteId: "site",
        fromEntityId: from,
        toEntityId: to,
        kind,
        weight,
        source: "user",
        status,
        reason,
        createdAt: "2026-09-19T10:00:00Z",
    };
}

export function pageType(preset: string, name = preset): PageType {
    return {
        id: `type-${preset}`,
        siteId: "site",
        name,
        preset,
        writable: preset === "hub" || preset === "entity" || preset === "product",
        templateId: null,
        pages: 0,
        scopes: [],
        createdAt: "2026-09-19T10:00:00Z",
        updatedAt: "2026-09-19T10:00:00Z",
    };
}

export function typedPage(entityId: string, preset: string | null, status = "planned"): EntityPage {
    return {
        entityId,
        pageId: `page-${entityId}`,
        path: `/${entityId}/`,
        status,
        pageTypeId: preset === null ? null : `type-${preset}`,
        work: "",
        mismatch: false,
    };
}

export const potteryEntities: Entity[] = [
    entity({ id: "pottery", name: "Pottery", score: 1 }),
    entity({ id: "teapots", name: "Teapots", score: 0.6 }),
    entity({ id: "care", name: "Care Guide", score: 0.3, primaryKeyword: "mug care guide" }),
    entity({ id: "mugs", name: "Ceramic Mugs", score: 0.9 }),
    entity({ id: "glazing", name: "Glazing", score: 0.7 }),
    entity({ id: "m350", name: "Mug 350ml", score: 0.5 }),
    entity({ id: "m500", name: "Mug 500ml", score: 0.4 }),
    entity({ id: "travel", name: "Travel Mug", score: 0.2, page: false, source: "ai" }),
    entity({ id: "stone", name: "Stoneware", score: 0.82, secondaryKeywords: ["stoneware mugs"] }),
    entity({ id: "kiln", name: "Kiln Care", score: 0.35, page: false, source: "ai" }),
];

export const potteryTypes: PageType[] = [
    pageType("hub", "Hub"),
    pageType("entity", "Guide"),
    pageType("product", "Product"),
    pageType("store_category", "Store category"),
];

export const potteryPages: EntityPage[] = [
    typedPage("pottery", "hub"),
    typedPage("teapots", "hub"),
    typedPage("care", "entity"),
    typedPage("mugs", "hub"),
    typedPage("glazing", "entity"),
    typedPage("m350", "product"),
    typedPage("m500", "product"),
    typedPage("stone", "store_category"),
];

export const potteryEdges: Edge[] = [
    edge("e1", "mugs", "pottery", "parent"),
    edge("e2", "glazing", "pottery", "parent"),
    edge("e3", "m350", "mugs", "parent"),
    edge("e4", "m500", "mugs", "parent"),
    edge("e5", "travel", "mugs", "parent"),
    edge("e6", "stone", "mugs", "parent"),
    edge("e7", "stone", "pottery", "parent"),
    edge("e8", "kiln", "glazing", "parent", "proposed", 1, "kiln care sits under glazing"),
    edge("e9", "glazing", "mugs", "related", "approved", 0.62),
    edge("e10", "mugs", "teapots", "related", "proposed", 0.86, "both are stoneware tableware"),
    edge("e11", "kiln", "stone", "related", "proposed", 0.71),
    edge("e12", "care", "pottery", "parent", "rejected"),
];
