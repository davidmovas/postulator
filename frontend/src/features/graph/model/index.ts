import type { Edge, Entity, EntityPage } from "../../../data/types.js";

export type NodeState = "mismatch" | "working" | "published" | "exists" | "planned" | "archived" | "noPage";

export const nodeStates: readonly NodeState[] = [
    "mismatch",
    "working",
    "published",
    "exists",
    "planned",
    "archived",
    "noPage",
];

export interface RelatedLink {
    edgeId: string;
    otherId: string;
    weight: number;
    status: string;
}

export interface EntityProblems {
    noPage: boolean;
    orphan: boolean;
    proposed: number;
    multiParent: boolean;
}

export interface GraphCounts {
    total: number;
    noPage: number;
    orphan: number;
    proposedEdges: number;
    ai: number;
    multiParent: number;
    states: Readonly<Record<NodeState, number>>;
}

export interface GraphIndex {
    entities: readonly Entity[];
    edges: readonly Edge[];
    byId: ReadonlyMap<string, Entity>;
    edgeById: ReadonlyMap<string, Edge>;
    placementParent: ReadonlyMap<string, string>;
    placementProposed: ReadonlySet<string>;
    children: ReadonlyMap<string, readonly string[]>;
    roots: readonly string[];
    approvedParents: ReadonlyMap<string, readonly string[]>;
    proposedParents: ReadonlyMap<string, readonly string[]>;
    related: ReadonlyMap<string, readonly RelatedLink[]>;
    proposedEdges: readonly Edge[];
    subtreeCount: ReadonlyMap<string, number>;
    depth: ReadonlyMap<string, number>;
    problems: ReadonlyMap<string, EntityProblems>;
    pageState: ReadonlyMap<string, EntityPage>;
    counts: GraphCounts;
}

export function nodeStateOf(index: GraphIndex, id: string): NodeState {
    const page = index.pageState.get(id);
    if (page === undefined) {
        return "noPage";
    }
    if (page.mismatch) {
        return "mismatch";
    }
    if (page.work !== "") {
        return "working";
    }
    switch (page.status) {
        case "published":
        case "exists":
        case "planned":
        case "archived":
            return page.status;
        default:
            return "noPage";
    }
}

export function compareNames(left: string, right: string): number {
    const a = left.toLowerCase();
    const b = right.toLowerCase();
    if (a === b) {
        return 0;
    }
    return a < b ? -1 : 1;
}

export function byScore(byId: ReadonlyMap<string, Entity>): (left: string, right: string) => number {
    return (left, right) => {
        const a = byId.get(left);
        const b = byId.get(right);
        if (a === undefined || b === undefined) {
            return 0;
        }
        if (a.score !== b.score) {
            return b.score - a.score;
        }
        const named = compareNames(a.name, b.name);
        return named !== 0 ? named : compareNames(a.id, b.id);
    };
}

export function byName(byId: ReadonlyMap<string, Entity>): (left: string, right: string) => number {
    return (left, right) => {
        const a = byId.get(left);
        const b = byId.get(right);
        if (a === undefined || b === undefined) {
            return 0;
        }
        const named = compareNames(a.name, b.name);
        return named !== 0 ? named : compareNames(a.id, b.id);
    };
}

function push<T>(held: Map<string, T[]>, key: string, value: T): void {
    const list = held.get(key);
    if (list === undefined) {
        held.set(key, [value]);
    } else {
        list.push(value);
    }
}

function breakCycles(placement: Map<string, string>, proposed: Set<string>): void {
    const state = new Map<string, 1 | 2>();
    for (const start of placement.keys()) {
        const path: string[] = [];
        let current: string | undefined = start;
        while (current !== undefined) {
            const seen = state.get(current);
            if (seen === 2) {
                break;
            }
            if (seen === 1) {
                const cycle = path.slice(path.indexOf(current));
                const weakest = cycle.find((member) => proposed.has(member)) ?? current;
                placement.delete(weakest);
                proposed.delete(weakest);
                break;
            }
            state.set(current, 1);
            path.push(current);
            current = placement.get(current);
        }
        for (const member of path) {
            state.set(member, 2);
        }
    }
}

export function buildGraphIndex(
    entities: readonly Entity[],
    edges: readonly Edge[],
    pages: readonly EntityPage[] = [],
): GraphIndex {
    const pageState = new Map<string, EntityPage>();
    for (const state of pages) {
        const held = pageState.get(state.entityId);
        if (held === undefined || rank(state) > rank(held)) {
            pageState.set(state.entityId, state);
        }
    }

    const byId = new Map<string, Entity>();
    for (const held of entities) {
        byId.set(held.id, held);
    }

    const edgeById = new Map<string, Edge>();
    const approvedParents = new Map<string, string[]>();
    const proposedParents = new Map<string, string[]>();
    const related = new Map<string, RelatedLink[]>();
    const proposedTouches = new Map<string, number>();
    const proposedEdges: Edge[] = [];
    for (const held of edges) {
        if (held.status === "rejected" || !byId.has(held.fromEntityId) || !byId.has(held.toEntityId)) {
            continue;
        }
        edgeById.set(held.id, held);
        if (held.status === "proposed") {
            proposedEdges.push(held);
            proposedTouches.set(held.fromEntityId, (proposedTouches.get(held.fromEntityId) ?? 0) + 1);
            proposedTouches.set(held.toEntityId, (proposedTouches.get(held.toEntityId) ?? 0) + 1);
        }
        if (held.kind === "parent") {
            push(held.status === "approved" ? approvedParents : proposedParents, held.fromEntityId, held.toEntityId);
        } else if (held.kind === "related") {
            push(related, held.fromEntityId, { edgeId: held.id, otherId: held.toEntityId, weight: held.weight, status: held.status });
            push(related, held.toEntityId, { edgeId: held.id, otherId: held.fromEntityId, weight: held.weight, status: held.status });
        }
    }

    const scoreOrder = byScore(byId);
    for (const list of approvedParents.values()) {
        list.sort(scoreOrder);
    }
    for (const list of proposedParents.values()) {
        list.sort(scoreOrder);
    }
    for (const list of related.values()) {
        list.sort((left, right) => (left.weight !== right.weight ? right.weight - left.weight : compareNames(left.edgeId, right.edgeId)));
    }
    proposedEdges.sort((left, right) => {
        if (left.weight !== right.weight) {
            return right.weight - left.weight;
        }
        const created = compareNames(left.createdAt ?? "", right.createdAt ?? "");
        return created !== 0 ? created : compareNames(left.id, right.id);
    });

    const placementParent = new Map<string, string>();
    const placementProposed = new Set<string>();
    for (const id of byId.keys()) {
        const approved = approvedParents.get(id);
        if (approved !== undefined && approved.length > 0) {
            placementParent.set(id, approved[0]);
            continue;
        }
        const proposed = proposedParents.get(id);
        if (proposed !== undefined && proposed.length > 0) {
            placementParent.set(id, proposed[0]);
            placementProposed.add(id);
        }
    }
    breakCycles(placementParent, placementProposed);

    const children = new Map<string, string[]>();
    const roots: string[] = [];
    for (const id of byId.keys()) {
        const parent = placementParent.get(id);
        if (parent === undefined) {
            roots.push(id);
        } else {
            push(children, parent, id);
        }
    }
    roots.sort(scoreOrder);
    for (const list of children.values()) {
        list.sort(scoreOrder);
    }

    const depth = new Map<string, number>();
    const subtreeCount = new Map<string, number>();
    const order: string[] = [];
    const stack: Array<{ id: string; depth: number }> = roots.map((id) => ({ id, depth: 0 })).reverse();
    while (stack.length > 0) {
        const next = stack.pop();
        if (next === undefined) {
            break;
        }
        depth.set(next.id, next.depth);
        order.push(next.id);
        const below = children.get(next.id) ?? [];
        for (let index = below.length - 1; index >= 0; index -= 1) {
            stack.push({ id: below[index], depth: next.depth + 1 });
        }
    }
    for (let index = order.length - 1; index >= 0; index -= 1) {
        const id = order[index];
        let total = 0;
        for (const child of children.get(id) ?? []) {
            total += 1 + (subtreeCount.get(child) ?? 0);
        }
        subtreeCount.set(id, total);
    }

    const problems = new Map<string, EntityProblems>();
    const states: Record<NodeState, number> = {
        mismatch: 0, working: 0, published: 0, exists: 0, planned: 0, archived: 0, noPage: 0,
    };
    const counts: GraphCounts = {
        total: byId.size, noPage: 0, orphan: 0, proposedEdges: proposedEdges.length, ai: 0, multiParent: 0,
        states,
    };
    for (const held of byId.values()) {
        const approved = approvedParents.get(held.id) ?? [];
        const flags: EntityProblems = {
            noPage: held.canonicalPageId === null,
            orphan: approved.length === 0 && held.kind !== "hub",
            proposed: proposedTouches.get(held.id) ?? 0,
            multiParent: approved.length > 1,
        };
        problems.set(held.id, flags);
        counts.noPage += flags.noPage ? 1 : 0;
        counts.orphan += flags.orphan ? 1 : 0;
        counts.multiParent += flags.multiParent ? 1 : 0;
        counts.ai += held.source === "ai" ? 1 : 0;
    }

    const built: GraphIndex = {
        entities,
        edges,
        byId,
        edgeById,
        placementParent,
        placementProposed,
        children,
        roots,
        approvedParents,
        proposedParents,
        related,
        proposedEdges,
        subtreeCount,
        depth,
        problems,
        pageState,
        counts,
    };
    for (const id of byId.keys()) {
        states[nodeStateOf(built, id)] += 1;
    }
    return built;
}

function rank(state: EntityPage): number {
    if (state.mismatch) {
        return 3;
    }
    if (state.work !== "") {
        return 2;
    }
    return state.status === "published" ? 1 : 0;
}
