import type { ExtraLink, LinkAuditPage, RequiredLink } from "../../../data/types.js";

export const extraOrder = ["unknown_internal", "self", "external"] as const;

export interface ExtraGroup {
    kind: string;
    links: ExtraLink[];
}

export interface DetailCounts {
    satisfied: number;
    missing: number;
    missingRequired: number;
    blocked: number;
    offGraph: number;
    anchorNotAllowed: number;
}

export interface DetailGroups {
    up: RequiredLink[];
    down: RequiredLink[];
    sibling: RequiredLink[];
    blocked: RequiredLink[];
    extra: ExtraGroup[];
    counts: DetailCounts;
}

export function groups(detail: LinkAuditPage): DetailGroups {
    const required = detail.required ?? [];
    const extra = detail.extra ?? [];
    const out: DetailGroups = {
        up: [],
        down: [],
        sibling: [],
        blocked: [],
        extra: [],
        counts: { satisfied: 0, missing: 0, missingRequired: 0, blocked: 0, offGraph: extra.length, anchorNotAllowed: 0 },
    };
    for (const link of required) {
        if (link.blockedReason !== "") {
            out.blocked.push(link);
            out.counts.blocked += 1;
            continue;
        }
        if (link.relation === "up") {
            out.up.push(link);
        } else if (link.relation === "down") {
            out.down.push(link);
        } else {
            out.sibling.push(link);
        }
        if (link.satisfied) {
            out.counts.satisfied += 1;
            if (!link.anchorAllowed) {
                out.counts.anchorNotAllowed += 1;
            }
        } else {
            out.counts.missing += 1;
            if (link.required) {
                out.counts.missingRequired += 1;
            }
        }
    }
    const byKind = new Map<string, ExtraLink[]>();
    for (const link of extra) {
        const held = byKind.get(link.kind);
        if (held === undefined) {
            byKind.set(link.kind, [link]);
        } else {
            held.push(link);
        }
    }
    for (const kind of extraOrder) {
        const links = byKind.get(kind);
        if (links !== undefined) {
            out.extra.push({ kind, links });
            byKind.delete(kind);
        }
    }
    for (const [kind, links] of byKind) {
        out.extra.push({ kind, links });
    }
    return out;
}
