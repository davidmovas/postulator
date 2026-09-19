import type { Page, PageTreeNode } from "../../data/types.js";

export interface TreeBranch {
    kind: "page";
    id: string;
    page: Page;
    depth: number;
    childCount: number;
    expanded: boolean;
}

export interface TreeMore {
    kind: "more";
    id: string;
    parentId: string;
    depth: number;
    hidden: number;
}

export type TreeRow = TreeBranch | TreeMore;

export const rootParentId = "";

function nodesOf(node: PageTreeNode): readonly PageTreeNode[] {
    return node.children ?? [];
}

export function countNodes(roots: readonly PageTreeNode[] | null | undefined): number {
    if (roots === null || roots === undefined) {
        return 0;
    }
    let total = 0;
    const stack: PageTreeNode[] = [...roots];
    while (stack.length > 0) {
        const node = stack.pop();
        if (node === undefined) {
            break;
        }
        total += 1;
        for (const child of nodesOf(node)) {
            stack.push(child);
        }
    }
    return total;
}

export function branchIds(roots: readonly PageTreeNode[] | null | undefined): string[] {
    if (roots === null || roots === undefined) {
        return [];
    }
    const ids: string[] = [];
    const stack: PageTreeNode[] = [...roots];
    while (stack.length > 0) {
        const node = stack.pop();
        if (node === undefined) {
            break;
        }
        const children = nodesOf(node);
        if (children.length > 0) {
            ids.push(node.page.id);
        }
        for (const child of children) {
            stack.push(child);
        }
    }
    return ids;
}

export function pathTo(roots: readonly PageTreeNode[] | null | undefined, pageId: string): string[] {
    const trail: string[] = [];
    function descend(nodes: readonly PageTreeNode[]): boolean {
        for (const node of nodes) {
            if (node.page.id === pageId) {
                return true;
            }
            trail.push(node.page.id);
            if (descend(nodesOf(node))) {
                return true;
            }
            trail.pop();
        }
        return false;
    }
    if (roots === null || roots === undefined) {
        return [];
    }
    return descend(roots) ? trail : [];
}

function walk(
    nodes: readonly PageTreeNode[],
    parentId: string,
    depth: number,
    expanded: ReadonlySet<string>,
    lifted: ReadonlySet<string>,
    childLimit: number,
    out: TreeRow[],
): void {
    const limit = lifted.has(parentId) ? nodes.length : Math.min(childLimit, nodes.length);
    for (let index = 0; index < limit; index += 1) {
        const node = nodes[index];
        const children = nodesOf(node);
        const open = children.length > 0 && expanded.has(node.page.id);
        out.push({
            kind: "page",
            id: node.page.id,
            page: node.page,
            depth,
            childCount: children.length,
            expanded: open,
        });
        if (open) {
            walk(children, node.page.id, depth + 1, expanded, lifted, childLimit, out);
        }
    }
    const hidden = nodes.length - limit;
    if (hidden > 0) {
        out.push({ kind: "more", id: `more:${parentId}`, parentId, depth, hidden });
    }
}

export function flattenTree(
    roots: readonly PageTreeNode[] | null | undefined,
    expanded: ReadonlySet<string>,
    lifted: ReadonlySet<string>,
    childLimit: number,
): TreeRow[] {
    if (roots === null || roots === undefined || roots.length === 0) {
        return [];
    }
    const out: TreeRow[] = [];
    walk(roots, rootParentId, 0, expanded, lifted, childLimit, out);
    return out;
}
