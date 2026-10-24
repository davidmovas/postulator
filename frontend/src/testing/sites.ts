import type { Site, SiteSettings } from "../data/types.js";

export function siteSettings(overrides: Partial<SiteSettings> = {}): SiteSettings {
    return {
        linkUrlForm: "auto" as SiteSettings["linkUrlForm"],
        userAgent: "",
        observe: { renderConcurrency: 4, fullWalkEvery: 10 },
        builders: [{ name: "elementor", dataAttributes: ["data-elementor-type"], classPrefixes: ["elementor-"], shortcodePrefixes: [] }],
        technicalShortcodes: ["woocommerce_cart"],
        technicalBlocks: ["woocommerce/cart"],
        technical: { front: 2, posts: 0, privacy: 3, shop: 5, cart: 6, checkout: 7, account: 8, terms: 0 },
        observation: {
            lastStartedAt: "2026-10-18T06:58:00Z",
            lastFinishedAt: "2026-10-18T07:00:00Z",
            lastFullWalkAt: "2026-10-12T07:00:00Z",
            lastModifiedSeen: "2026-10-17T21:00:00Z",
            sinceFullWalk: 3,
        },
        ...overrides,
    };
}

export function aSite(overrides: Partial<Site> = {}): Site {
    return {
        id: "s1",
        name: "Peptide shop",
        baseUrl: "https://shop.test",
        username: "editor",
        status: "active",
        allowInsecure: false,
        plugin: { installed: true, version: "1.4.1", capabilities: ["raw", "preview"], seoPlugin: "rankmath" },
        commerce: "ready",
        defaults: { templateId: null, modelProfiles: null },
        settings: siteSettings(),
        createdAt: "2026-10-01T09:00:00Z",
        updatedAt: "2026-10-18T07:00:00Z",
        ...overrides,
    };
}
