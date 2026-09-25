import type { ExtraLink, LinkAuditPage, PageAudit, RequiredLink } from "../../../data/types.js";

export interface RowSeed {
    pageId: string;
    path: string;
    status?: string;
    entityId?: string;
    entityName?: string;
    skipReason?: string;
    targets?: number;
    required?: number;
    satisfied?: number;
    missing?: number;
    missingRequired?: number;
    blocked?: number;
    offGraph?: number;
    pending?: number;
    unpublished?: number;
    inbound?: number;
    orphan?: boolean;
    onSite?: boolean;
}

export function row(seed: RowSeed): PageAudit {
    return {
        pageId: seed.pageId,
        path: seed.path,
        status: seed.status ?? "published",
        entityId: seed.entityId ?? "",
        entityName: seed.entityName ?? "",
        skipReason: seed.skipReason ?? "",
        onSite: seed.onSite ?? (seed.status ?? "published") !== "planned",
        targets: seed.targets ?? 0,
        required: seed.required ?? 0,
        satisfied: seed.satisfied ?? 0,
        missing: seed.missing ?? 0,
        missingRequired: seed.missingRequired ?? 0,
        blocked: seed.blocked ?? 0,
        offGraph: seed.offGraph ?? 0,
        pending: seed.pending ?? 0,
        unpublished: seed.unpublished ?? 0,
        inbound: seed.inbound ?? 0,
        orphan: seed.orphan ?? false,
    };
}

export const auditRows: PageAudit[] = [
    row({ pageId: "p-pottery", path: "/pottery/", entityId: "pottery", entityName: "Pottery", targets: 3, satisfied: 2, missing: 1, inbound: 4 }),
    row({
        pageId: "p-mugs",
        path: "/pottery/mugs/",
        entityId: "mugs",
        entityName: "Ceramic Mugs",
        targets: 5,
        required: 1,
        satisfied: 2,
        missing: 2,
        missingRequired: 1,
        blocked: 1,
        offGraph: 2,
        inbound: 2,
    }),
    row({ pageId: "p-m350", path: "/pottery/mugs/350/", entityId: "m350", entityName: "Mug 350ml", targets: 2, required: 1, satisfied: 2, inbound: 1 }),
    row({ pageId: "p-glazing", path: "/pottery/glazing/", entityId: "glazing", entityName: "Glazing", targets: 2, required: 1, satisfied: 1, missing: 1, orphan: true }),
    row({ pageId: "p-blog", path: "/blog/diary/", status: "exists", skipReason: "unmapped", offGraph: 0, inbound: 0, orphan: true }),
    row({ pageId: "p-tea", path: "/tea/", entityId: "tea", entityName: "Tea", skipReason: "no_template", inbound: 1 }),
];

export function required(seed: Partial<RequiredLink> & { relation: string; targetEntityId: string }): RequiredLink {
    return {
        relation: seed.relation,
        required: seed.required ?? false,
        targetEntityId: seed.targetEntityId,
        targetEntityName: seed.targetEntityName ?? seed.targetEntityId,
        targetPageId: seed.targetPageId ?? `page-${seed.targetEntityId}`,
        targetPath: seed.targetPath ?? `/${seed.targetEntityId}/`,
        satisfied: seed.satisfied ?? false,
        anchor: seed.anchor ?? "",
        anchorAllowed: seed.anchorAllowed ?? false,
        anchorsAllowed: seed.anchorsAllowed ?? [seed.targetEntityId],
        weight: seed.weight ?? 1,
        depth: seed.depth ?? 1,
        blockedReason: seed.blockedReason ?? "",
        state: seed.state ?? ((seed.blockedReason ?? "") !== "" ? "blocked" : (seed.satisfied ?? false) ? "placed" : "missing"),
        targetOnSite: seed.targetOnSite ?? true,
    };
}

export function extra(seed: Partial<ExtraLink> & { kind: string }): ExtraLink {
    return {
        toUrl: seed.toUrl ?? "/x/",
        toPageId: seed.toPageId ?? "",
        anchor: seed.anchor ?? "x",
        kind: seed.kind,
        origin: seed.origin ?? "observed",
    };
}

export const mugsDetail: LinkAuditPage = {
    page: auditRows[1],
    templateId: "tpl-hub",
    rules: { upDepth: 1, downLinks: true, siblingMinWeight: 0.5, maxLinks: 20, maxPerTarget: 1, parentLinkWithinParagraphs: 2, childrenSection: true },
    required: [
        required({ relation: "up", required: true, targetEntityId: "pottery", satisfied: false }),
        required({ relation: "down", targetEntityId: "m350", satisfied: true, anchor: "350 ml mug", anchorAllowed: true, weight: 0.5 }),
        required({ relation: "down", targetEntityId: "m500", satisfied: true, anchor: "/mugs/500/", anchorAllowed: false, weight: 0.4 }),
        required({ relation: "sibling", targetEntityId: "glazing", satisfied: false, weight: 0.62 }),
        required({ relation: "down", targetEntityId: "travel", targetPageId: "", targetPath: "", blockedReason: "no_canonical_page", weight: 0.2 }),
    ],
    extra: [
        extra({ kind: "external", toUrl: "https://example.org/", anchor: "elsewhere" }),
        extra({ kind: "unknown_internal", toUrl: "/blog/diary/", toPageId: "p-blog", anchor: "diary" }),
        extra({ kind: "self", toUrl: "/pottery/mugs/", toPageId: "p-mugs", anchor: "here" }),
    ],
};

export const plannedRow: PageAudit = row({
    pageId: "p-cups",
    path: "/pottery/cups/",
    status: "planned",
    entityId: "cups",
    entityName: "Cups",
    targets: 3,
    required: 1,
    pending: 3,
});

export const plannedDetail: LinkAuditPage = {
    page: plannedRow,
    templateId: "tpl-hub",
    rules: mugsDetail.rules,
    required: [
        required({ relation: "up", required: true, targetEntityId: "pottery", state: "awaiting_page" }),
        required({ relation: "down", targetEntityId: "espresso-cup", state: "awaiting_page", targetOnSite: false }),
        required({ relation: "sibling", targetEntityId: "glazing", state: "awaiting_page" }),
    ],
    extra: [],
};
