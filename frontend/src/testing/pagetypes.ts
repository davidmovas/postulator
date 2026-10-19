import type { FieldScope, LinkPolicy, PageType } from "../data/types.js";
import { noLinks } from "../features/settings/types/model.js";

export function hubLinks(): LinkPolicy {
    return {
        ...noLinks(),
        links: true,
        rules: {
            upDepth: 1,
            downLinks: true,
            siblingMinWeight: 0.5,
            maxLinks: 20,
            maxPerTarget: 1,
            parentLinkWithinParagraphs: 2,
            childrenSection: true,
        },
        forbidExternal: true,
        forbidSelf: true,
    };
}

export function aScope(field: string, overrides: Partial<FieldScope> = {}): FieldScope {
    return {
        field,
        label: "",
        provider: "",
        enabled: false,
        aiWrites: true,
        instruction: "",
        minLength: 0,
        maxLength: 0,
        rewrites: false,
        links: noLinks(),
        stored: false,
        ...overrides,
    };
}

export function aPageType(id: string, preset: string, overrides: Partial<PageType> = {}): PageType {
    return {
        id,
        siteId: "s1",
        name: preset,
        preset,
        writable: preset === "hub" || preset === "entity" || preset === "product",
        templateId: null,
        pages: 0,
        scopes: [],
        createdAt: "2026-10-18T09:00:00Z",
        updatedAt: "2026-10-18T09:00:00Z",
        ...overrides,
    };
}
