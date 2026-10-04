import { copy } from "../../copy/index.js";
import type { CategoryNode } from "../../data/types.js";

export interface CategoryTree {
    byId: ReadonlyMap<string, CategoryNode>;
    roots: readonly string[];
    children: ReadonlyMap<string, readonly string[]>;
}

export interface CategoryRow {
    node: CategoryNode;
    depth: number;
    branches: boolean;
    open: boolean;
}

export interface CategoryChip {
    state: "onSite" | "onPublish";
    label: string;
    hint: string;
}

export function categoryTree(nodes: readonly CategoryNode[]): CategoryTree {
    const byId = new Map<string, CategoryNode>();
    for (const node of nodes) {
        byId.set(node.id, node);
    }
    const roots: string[] = [];
    const children = new Map<string, string[]>();
    for (const node of nodes) {
        const parent = node.parentId;
        if (parent === null || parent === "" || !byId.has(parent)) {
            roots.push(node.id);
            continue;
        }
        const under = children.get(parent);
        if (under === undefined) {
            children.set(parent, [node.id]);
        } else {
            under.push(node.id);
        }
    }
    return { byId, roots, children };
}

export function categoryRows(tree: CategoryTree, open: ReadonlySet<string>): readonly CategoryRow[] {
    const rows: CategoryRow[] = [];
    const seen = new Set<string>();
    const walk = (ids: readonly string[], depth: number): void => {
        for (const id of ids) {
            const node = tree.byId.get(id);
            if (node === undefined || seen.has(id)) {
                continue;
            }
            seen.add(id);
            const below = tree.children.get(id) ?? [];
            const expanded = below.length > 0 && open.has(id);
            rows.push({ node, depth, branches: below.length > 0, open: expanded });
            if (expanded) {
                walk(below, depth + 1);
            }
        }
    };
    walk(tree.roots, 0);
    return rows;
}

export function ancestorsOf(tree: CategoryTree, id: string): readonly string[] {
    const out: string[] = [];
    const seen = new Set<string>([id]);
    let parent = tree.byId.get(id)?.parentId ?? null;
    while (parent !== null && parent !== "" && tree.byId.has(parent) && !seen.has(parent)) {
        seen.add(parent);
        out.unshift(parent);
        parent = tree.byId.get(parent)?.parentId ?? null;
    }
    return out;
}

function termOf(held: number | null | undefined): number | null {
    return typeof held === "number" && Number.isFinite(held) && held > 0 ? held : null;
}

export function categoryChip(node: CategoryNode): CategoryChip {
    const said = copy.pages.categories;
    const page = termOf(node.termIds.category);
    const product = termOf(node.termIds.productCategory);
    if (page !== null && product !== null) {
        return { state: "onSite", label: said.termLabel(page), hint: said.onSiteBoth(page, product) };
    }
    if (page !== null) {
        return { state: "onSite", label: said.termLabel(page), hint: said.onSite(page) };
    }
    if (product !== null) {
        return { state: "onSite", label: said.termLabel(product), hint: said.onSiteProduct(product) };
    }
    return { state: "onPublish", label: said.newLabel, hint: said.onPublish };
}
