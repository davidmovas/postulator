import type { GraphIndex } from "./index.js";

export const lenses = ["all", "noPage", "proposed", "orphan", "ai"] as const;

export type Lens = (typeof lenses)[number];

export function isLens(value: string): value is Lens {
    return (lenses as readonly string[]).includes(value);
}

function matchesLens(index: GraphIndex, lens: Lens, id: string): boolean {
    const flags = index.problems.get(id);
    switch (lens) {
        case "all":
            return true;
        case "noPage":
            return flags?.noPage === true;
        case "proposed":
            return (flags?.proposed ?? 0) > 0;
        case "orphan":
            return flags?.orphan === true;
        case "ai":
            return index.byId.get(id)?.source === "ai";
    }
}

export function matchedSet(index: GraphIndex, lens: Lens, kinds: ReadonlySet<string> | null): ReadonlySet<string> {
    const out = new Set<string>();
    const anyKind = kinds === null || kinds.size === 0;
    for (const held of index.byId.values()) {
        if ((anyKind || kinds.has(held.kind)) && matchesLens(index, lens, held.id)) {
            out.add(held.id);
        }
    }
    return out;
}

export function lensCounts(index: GraphIndex): Record<Lens, number> {
    const counts: Record<Lens, number> = { all: 0, noPage: 0, proposed: 0, orphan: 0, ai: 0 };
    for (const id of index.byId.keys()) {
        for (const lens of lenses) {
            if (matchesLens(index, lens, id)) {
                counts[lens] += 1;
            }
        }
    }
    return counts;
}

export function isolate(index: GraphIndex, matched: ReadonlySet<string>): ReadonlySet<string> {
    const kept = new Set<string>();
    for (const id of matched) {
        let current: string | undefined = id;
        while (current !== undefined && !kept.has(current)) {
            kept.add(current);
            current = index.placementParent.get(current);
        }
    }
    return kept;
}
