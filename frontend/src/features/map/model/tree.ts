import type { SiteMapNode } from "../../../data/types.js";

export interface Branch {
    nodes: readonly SiteMapNode[];
    hasMore: boolean;
    pending: boolean;
}

export type Branches = ReadonlyMap<string, Branch>;

export type TreeRow =
    | { kind: "node"; key: string; node: SiteMapNode; depth: number; expanded: boolean }
    | { kind: "more"; key: string; parentId: string; depth: number }
    | { kind: "loading"; key: string; parentId: string; depth: number };

function opens(node: SiteMapNode, expanded: ReadonlySet<string>): boolean {
    return node.children > 0 && expanded.has(node.page.pageId);
}

export function openBranches(root: string, expanded: ReadonlySet<string>, branches: Branches): string[] {
    const out = [root];
    const seen = new Set([root]);
    const walk = (parentId: string): void => {
        for (const node of branches.get(parentId)?.nodes ?? []) {
            const id = node.page.pageId;
            if (opens(node, expanded) && !seen.has(id)) {
                seen.add(id);
                out.push(id);
                walk(id);
            }
        }
    };
    walk(root);
    return out;
}

export function flattenTree(root: string, expanded: ReadonlySet<string>, branches: Branches): TreeRow[] {
    const out: TreeRow[] = [];
    const seen = new Set([root]);
    const walk = (parentId: string, depth: number): void => {
        const branch = branches.get(parentId);
        if (branch === undefined || (branch.pending && branch.nodes.length === 0)) {
            out.push({ kind: "loading", key: `loading:${parentId}`, parentId, depth });
            return;
        }
        for (const node of branch.nodes) {
            const id = node.page.pageId;
            const open = opens(node, expanded) && !seen.has(id);
            out.push({ kind: "node", key: `node:${id}`, node, depth, expanded: open });
            if (open) {
                seen.add(id);
                walk(id, depth + 1);
            }
        }
        if (branch.hasMore) {
            out.push({ kind: "more", key: `more:${parentId}`, parentId, depth });
        }
    };
    walk(root, 0);
    return out;
}

export function toggled(expanded: ReadonlySet<string>, id: string): ReadonlySet<string> {
    const next = new Set(expanded);
    if (next.has(id)) {
        next.delete(id);
    } else {
        next.add(id);
    }
    return next;
}
