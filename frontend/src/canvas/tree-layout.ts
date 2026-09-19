import type { Rect } from "./viewport.js";

export interface TreeInput {
    id: string;
    width: number;
    children: readonly TreeInput[];
}

export interface Placed {
    id: string;
    x: number;
    y: number;
    width: number;
    height: number;
    depth: number;
    parentId: string | null;
    row: number;
}

export interface Layout {
    nodes: readonly Placed[];
    byId: ReadonlyMap<string, Placed>;
    columns: readonly number[];
    bounds: Rect;
}

export interface LayoutOptions {
    rowHeight: number;
    nodeHeight: number;
    columnGap: number;
    rootGap: number;
}

interface Pending {
    id: string;
    width: number;
    depth: number;
    parentId: string | null;
    row: number;
}

function place(node: TreeInput, depth: number, parentId: string | null, next: { row: number }, out: Pending[], widths: number[]): number {
    widths[depth] = Math.max(widths[depth] ?? 0, node.width);
    const index = out.length;
    out.push({ id: node.id, width: node.width, depth, parentId, row: 0 });
    if (node.children.length === 0) {
        const row = next.row;
        next.row += 1;
        out[index].row = row;
        return row;
    }
    let first = Number.NaN;
    let last = 0;
    for (const child of node.children) {
        const row = place(child, depth + 1, node.id, next, out, widths);
        if (Number.isNaN(first)) {
            first = row;
        }
        last = row;
    }
    const row = (first + last) / 2;
    out[index].row = row;
    return row;
}

export function layoutTree(roots: readonly TreeInput[], options: LayoutOptions): Layout {
    if (roots.length === 0) {
        return { nodes: [], byId: new Map(), columns: [], bounds: { x: 0, y: 0, width: 0, height: 0 } };
    }

    const pending: Pending[] = [];
    const widths: number[] = [];
    const next = { row: 0 };
    for (let index = 0; index < roots.length; index += 1) {
        if (index > 0) {
            next.row += options.rootGap;
        }
        place(roots[index], 0, null, next, pending, widths);
    }

    const columns: number[] = [];
    let offset = 0;
    for (let depth = 0; depth < widths.length; depth += 1) {
        columns.push(offset);
        offset += widths[depth] + options.columnGap;
    }

    const nodes: Placed[] = [];
    const byId = new Map<string, Placed>();
    let right = 0;
    let bottom = 0;
    for (const entry of pending) {
        const placed: Placed = {
            id: entry.id,
            x: columns[entry.depth],
            y: entry.row * options.rowHeight,
            width: entry.width,
            height: options.nodeHeight,
            depth: entry.depth,
            parentId: entry.parentId,
            row: entry.row,
        };
        nodes.push(placed);
        byId.set(placed.id, placed);
        right = Math.max(right, placed.x + placed.width);
        bottom = Math.max(bottom, placed.y + placed.height);
    }

    return { nodes, byId, columns, bounds: { x: 0, y: 0, width: right, height: bottom } };
}
