import type { FieldScope, LinkPolicy, PageType } from "../data/types.js";
import type { PageTypeRole } from "../generated/vocab.js";
import { noLinks } from "../features/settings/types/model.js";

const presetRoles: Readonly<Record<string, PageTypeRole>> = {
    landing: "hub",
    category: "hub",
    hub: "hub",
    guide: "entity",
    info: "entity",
    entity: "entity",
    product: "product",
    store_category: "archive",
    catalog: "catalog",
    technical: "technical",
};

export function roleOfPreset(preset: string): PageTypeRole {
    return presetRoles[preset] ?? "entity";
}

function writableRole(role: PageTypeRole): boolean {
    return role === "hub" || role === "entity" || role === "product";
}

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
        role: roleOfPreset(preset),
        writable: writableRole(roleOfPreset(preset)),
        templateId: null,
        pages: 0,
        scopes: [],
        createdAt: "2026-10-18T09:00:00Z",
        updatedAt: "2026-10-18T09:00:00Z",
        ...overrides,
    };
}
