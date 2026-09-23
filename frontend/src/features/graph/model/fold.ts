import type { TreeInput } from "../../../canvas/tree-layout.js";
import type { GraphIndex } from "./index.js";
import { byName } from "./index.js";

export const childLimit = 50;
export const autoExpandLimit = 200;

export type SortOrder = "score" | "name";

export interface FoldState {
    expanded: ReadonlySet<string>;
    lifted: ReadonlyMap<string, number>;
}

export interface VisibleEntity {
    kind: "entity";
    id: string;
    depth: number;
    parentId: string | null;
    childCount: number;
    expanded: boolean;
    hiddenChildren: number;
}

export interface VisibleMore {
    kind: "more";
    id: string;
    parentId: string;
    depth: number;
    hidden: number;
    shown: number;
}

export type VisibleRow = VisibleEntity | VisibleMore;

export function moreId(parentId: string): string {
    return `more:${parentId}`;
}

function branches(index: GraphIndex, keep: (id: string) => boolean): Set<string> {
    const out = new Set<string>();
    for (const [id, below] of index.children) {
        if (below.length > 0 && keep(id)) {
            out.add(id);
        }
    }
    return out;
}

export function expandAll(index: GraphIndex): FoldState {
    return { expanded: branches(index, () => true), lifted: new Map() };
}

export function collapseToDepth(index: GraphIndex, depth: number): FoldState {
    return { expanded: branches(index, (id) => (index.depth.get(id) ?? 0) < depth), lifted: new Map() };
}

export function defaultFold(index: GraphIndex): FoldState {
    return index.counts.total <= autoExpandLimit ? expandAll(index) : collapseToDepth(index, 1);
}

export function toggle(fold: FoldState, id: string): FoldState {
    const expanded = new Set(fold.expanded);
    if (expanded.has(id)) {
        expanded.delete(id);
    } else {
        expanded.add(id);
    }
    return { expanded, lifted: fold.lifted };
}

function limitOf(fold: FoldState, parentId: string): number {
    return fold.lifted.get(parentId) ?? childLimit;
}

export function liftMore(fold: FoldState, parentId: string, count: number): FoldState {
    const lifted = new Map(fold.lifted);
    lifted.set(parentId, limitOf(fold, parentId) + count);
    return { expanded: fold.expanded, lifted };
}

function orderedChildren(index: GraphIndex, parentId: string, order: SortOrder): readonly string[] {
    const held = index.children.get(parentId) ?? [];
    return order === "score" ? held : [...held].sort(byName(index.byId));
}

function orderedRoots(index: GraphIndex, order: SortOrder): readonly string[] {
    return order === "score" ? index.roots : [...index.roots].sort(byName(index.byId));
}

export function visibleRows(index: GraphIndex, fold: FoldState, order: SortOrder): VisibleRow[] {
    const out: VisibleRow[] = [];
    const walk = (ids: readonly string[], parentId: string | null, depth: number): void => {
        const limit = parentId === null ? ids.length : Math.min(limitOf(fold, parentId), ids.length);
        for (let position = 0; position < limit; position += 1) {
            const id = ids[position];
            const below = index.children.get(id) ?? [];
            const open = below.length > 0 && fold.expanded.has(id);
            out.push({
                kind: "entity",
                id,
                depth,
                parentId,
                childCount: below.length,
                expanded: open,
                hiddenChildren: open ? 0 : (index.subtreeCount.get(id) ?? 0),
            });
            if (open) {
                walk(orderedChildren(index, id, order), id, depth + 1);
            }
        }
        if (parentId !== null && ids.length > limit) {
            out.push({ kind: "more", id: moreId(parentId), parentId, depth, hidden: ids.length - limit, shown: limit });
        }
    };
    walk(orderedRoots(index, order), null, 0);
    return out;
}

export function reveal(index: GraphIndex, fold: FoldState, id: string, order: SortOrder = "score"): FoldState {
    if (!index.byId.has(id)) {
        return fold;
    }
    const expanded = new Set(fold.expanded);
    const lifted = new Map(fold.lifted);
    let child = id;
    let parent = index.placementParent.get(child);
    while (parent !== undefined) {
        expanded.add(parent);
        const position = orderedChildren(index, parent, order).indexOf(child);
        const limit = lifted.get(parent) ?? childLimit;
        if (position >= limit) {
            lifted.set(parent, Math.ceil((position + 1) / childLimit) * childLimit);
        }
        child = parent;
        parent = index.placementParent.get(child);
    }
    return { expanded, lifted };
}

export function treeOf(rows: readonly VisibleRow[], widthOf: (row: VisibleRow) => number): TreeInput[] {
    const roots: TreeInput[] = [];
    const open: Array<{ id: string; width: number; children: TreeInput[] }> = [];
    for (const row of rows) {
        const node = { id: row.id, width: widthOf(row), children: [] as TreeInput[] };
        open.length = row.depth;
        if (row.depth === 0) {
            roots.push(node);
        } else {
            open[row.depth - 1]?.children.push(node);
        }
        open[row.depth] = node;
    }
    return roots;
}
