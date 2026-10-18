import type { ModelTrail, PageModel } from "../../../data/types.js";

const seen = "2026-10-18T07:00:00Z";

const jsonldTrail: ModelTrail = {
    source: "jsonld",
    crumbs: [
        { name: "Home", path: "/", href: "https://shop.test/", final: "", status: 0, targetId: "p-home", targetTitle: "Home", self: false },
        {
            name: "Peptides",
            path: "/category/peptides/",
            href: "https://shop.test/category/peptides/",
            final: "/peptides/",
            status: 301,
            targetId: "p-peptides",
            targetTitle: "Peptides",
            self: false,
        },
        { name: "TB-500", path: "/tb-500/", href: "https://shop.test/tb-500/", final: "", status: 0, targetId: "p-tb500", targetTitle: "TB-500", self: true },
    ],
};

const navTrail: ModelTrail = {
    source: "nav",
    crumbs: [
        { name: "Home", path: "/", href: "/", final: "", status: 0, targetId: "p-home", targetTitle: "Home", self: false },
        { name: "Healing Peptides", path: "", href: "", final: "", status: 0, targetId: "", targetTitle: "", self: false },
        { name: "Recovery", path: "/recovery/", href: "/recovery/", final: "", status: 404, targetId: "", targetTitle: "", self: false },
        { name: "TB-500", path: "/tb-500/", href: "/tb-500/", final: "", status: 0, targetId: "p-tb500", targetTitle: "TB-500", self: true },
    ],
};

export const hubModel: PageModel = {
    pageId: "p-tb500",
    path: "/tb-500/",
    title: "TB-500",
    wpType: "page",
    wpId: 412,
    status: "published",
    identity: {
        url: "https://shop.test/tb-500/",
        status: "publish",
        wpType: "page",
        wpParentId: 0,
        modifiedGmt: "2026-10-16T08:00:00Z",
        observedAt: seen,
    },
    hierarchy: {
        parentId: "p-peptides",
        parentPath: "/peptides/",
        parentTitle: "Peptides",
        reason: "jsonld_breadcrumb",
        method: "redirect",
    },
    technicalReason: "",
    readOnlyReason: "",
    terms: [{ taxonomy: "product_cat", name: "Healing Peptides", slug: "healing-peptides" }],
    render: {
        root: "found",
        wordsContent: 9,
        wordsGenerated: 240,
        builder: "",
        urlForm: "absolute",
        robots: "index, follow",
        noIndex: false,
        finalPath: "",
        regions: [
            { kind: "breadcrumb", inside: false, links: 2, words: 5, signal: "aria" },
            { kind: "content_dynamic", inside: true, links: 0, words: 12, signal: "" },
            { kind: "listing", inside: false, links: 6, words: 80, signal: "jsonld" },
            { kind: "navigation", inside: false, links: 14, words: 40, signal: "landmark" },
        ],
        trails: [jsonldTrail, navTrail],
        headings: [],
        extractedAt: seen,
    },
    links: [
        {
            type: "breadcrumb",
            region: "breadcrumb",
            source: "render",
            position: null,
            ours: false,
            toUrl: "/peptides/",
            toPageId: "p-peptides",
            toPath: "/peptides/",
            anchor: "Peptides",
        },
        {
            type: "listing",
            region: "listing",
            source: "render",
            position: null,
            ours: false,
            toUrl: "/product/bpc-157-vial/",
            toPageId: "p-vial",
            toPath: "/product/bpc-157-vial/",
            anchor: "BPC-157 vial",
        },
        {
            type: "contextual",
            region: "content",
            source: "raw",
            position: 1,
            ours: true,
            toUrl: "/bpc-157/",
            toPageId: "p-bpc",
            toPath: "/bpc-157/",
            anchor: "BPC-157",
        },
    ],
    bodyKind: "classic",
    blocks: [
        { index: 0, kind: "shortcode", name: "entity_products", excerpt: "", dynamic: true, ours: false },
        {
            index: 1,
            kind: "paragraph",
            name: "",
            excerpt: "TB-500 is studied alongside BPC-157 for tissue repair.",
            dynamic: false,
            ours: true,
        },
    ],
    findings: [
        {
            id: "f-unresolved",
            code: "parent_unresolved",
            severity: "warning",
            subject: { name: "Healing Peptides", path: "" },
            createdAt: seen,
        },
        {
            id: "f-ambiguous",
            code: "parent_ambiguous",
            severity: "warning",
            subject: { name: "Guides", candidates: ["/lab-guides/", "/reconstitution/guides/"] },
            createdAt: seen,
        },
    ],
};

export const plannedModel: PageModel = {
    ...hubModel,
    pageId: "p-cjc",
    path: "/cjc-1295/",
    title: "CJC-1295",
    wpId: null,
    status: "planned",
    identity: null,
    hierarchy: { parentId: "p-peptides", parentPath: "/peptides/", parentTitle: "Peptides", reason: "workbook", method: "" },
    terms: [],
    render: null,
    links: [],
    bodyKind: "",
    blocks: [],
    findings: [],
};

export const unrenderedModel: PageModel = {
    ...hubModel,
    pageId: "p-guide",
    path: "/lab-guides/",
    title: "Lab guides",
    wpId: 77,
    render: null,
    links: [],
    bodyKind: "blocks",
    blocks: [],
    findings: [],
};

export const frontModel: PageModel = {
    ...unrenderedModel,
    pageId: "p-home",
    path: "/",
    title: "Home",
    hierarchy: { parentId: "", parentPath: "", parentTitle: "", reason: "none", method: "" },
    technicalReason: "",
    readOnlyReason: "builder:elementor",
};
