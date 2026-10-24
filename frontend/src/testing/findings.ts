import type { Finding } from "../data/types.js";

export function aFinding(overrides: Partial<Finding> = {}): Finding {
    return {
        id: "f1",
        siteId: "s1",
        pageId: "p1",
        pagePath: "/mugs/",
        pageTitle: "Mugs",
        code: "field_changed_on_site",
        severity: "warning",
        subjectKey: "seo_description",
        subject: { key: "seo_description", baseHash: "a", currentHash: "b" },
        options: [
            { name: "accept_site", kind: "model" },
            { name: "rewrite_field", kind: "run" },
            { name: "ignore", kind: "model" },
        ],
        status: "open",
        createdAt: "2026-10-22T09:00:00Z",
        updatedAt: "2026-10-22T09:00:00Z",
        ...overrides,
    };
}
