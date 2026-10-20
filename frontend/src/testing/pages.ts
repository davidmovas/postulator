import type { Page } from "../data/types.js";

export function aPage(id: string, path: string, overrides: Partial<Page> = {}): Page {
    return {
        id,
        siteId: "s1",
        path,
        plannedPath: "",
        slug: "",
        parentPageId: null,
        wpType: "page",
        wpId: null,
        title: `Title ${id}`,
        h1: "",
        metaTitle: "",
        metaDescription: "",
        canonical: "",
        keywords: [],
        notes: [],
        status: "planned",
        entityId: null,
        templateId: null,
        pageTypeId: null,
        typeSource: "",
        typeReason: "",
        contentHash: "",
        wpModifiedAt: null,
        lastSyncedAt: null,
        drift: false,
        createdAt: "2026-09-24T09:00:00Z",
        updatedAt: "2026-09-24T09:00:00Z",
        ...overrides,
    };
}
