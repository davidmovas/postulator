import type { Page, PageTreeNode } from "../../data/types.js";

export interface PickIndex {
    byId: ReadonlyMap<string, Page>;
    parentOf: ReadonlyMap<string, string>;
    childrenOf: ReadonlyMap<string, readonly string[]>;
}

export type Tick = "on" | "off" | "some";

export type Scope = "all" | "missing" | "present";

export const scopes: readonly Scope[] = ["all", "missing", "present"];

function childrenOf(node: PageTreeNode): readonly PageTreeNode[] {
    return node.children ?? [];
}

export function indexTree(roots: readonly PageTreeNode[] | null | undefined): PickIndex {
    const byId = new Map<string, Page>();
    const parentOf = new Map<string, string>();
    const children = new Map<string, string[]>();
    const stack: PageTreeNode[] = [...(roots ?? [])];
    while (stack.length > 0) {
        const node = stack.pop();
        if (node === undefined) {
            break;
        }
        byId.set(node.page.id, node.page);
        const below = childrenOf(node);
        children.set(
            node.page.id,
            below.map((child) => child.page.id),
        );
        for (const child of below) {
            parentOf.set(child.page.id, node.page.id);
            stack.push(child);
        }
    }
    return { byId, parentOf, childrenOf: children };
}

export function onSite(page: Page): boolean {
    return page.wpId !== null;
}

export function pickable(page: Page): boolean {
    return page.entityId !== null && page.entityId !== "";
}

export function requiredParents(selected: ReadonlySet<string>, index: PickIndex): ReadonlyMap<string, string> {
    const required = new Map<string, string>();
    for (const id of selected) {
        const neededBy = index.byId.get(id)?.path ?? "";
        let parentId = index.parentOf.get(id);
        while (parentId !== undefined) {
            const parent = index.byId.get(parentId);
            if (parent === undefined || onSite(parent)) {
                break;
            }
            if (!selected.has(parentId) && !required.has(parentId)) {
                required.set(parentId, neededBy);
            }
            parentId = index.parentOf.get(parentId);
        }
    }
    return required;
}

export function branchOf(id: string, index: PickIndex): string[] {
    const out: string[] = [];
    const stack = [id];
    while (stack.length > 0) {
        const next = stack.shift();
        if (next === undefined) {
            break;
        }
        const page = index.byId.get(next);
        if (page !== undefined && pickable(page)) {
            out.push(next);
        }
        stack.push(...(index.childrenOf.get(next) ?? []));
    }
    return out;
}

export function tick(
    id: string,
    selected: ReadonlySet<string>,
    required: ReadonlyMap<string, string>,
    index: PickIndex,
): Tick {
    if (selected.has(id) || required.has(id)) {
        return "on";
    }
    const stack = [...(index.childrenOf.get(id) ?? [])];
    while (stack.length > 0) {
        const next = stack.pop();
        if (next === undefined) {
            break;
        }
        if (selected.has(next)) {
            return "some";
        }
        stack.push(...(index.childrenOf.get(next) ?? []));
    }
    return "off";
}

function inScope(page: Page, scope: Scope): boolean {
    switch (scope) {
        case "missing":
            return !onSite(page);
        case "present":
            return onSite(page);
        default:
            return true;
    }
}

function matches(page: Page, query: string, scope: Scope): boolean {
    if (!inScope(page, scope)) {
        return false;
    }
    return query === "" || page.path.toLowerCase().includes(query) || page.title.toLowerCase().includes(query);
}

function narrowed(node: PageTreeNode, query: string, scope: Scope): PageTreeNode | null {
    const kept = childrenOf(node)
        .map((child) => narrowed(child, query, scope))
        .filter((child): child is PageTreeNode => child !== null);
    if (kept.length > 0 || matches(node.page, query, scope)) {
        return { page: node.page, children: kept };
    }
    return null;
}

export function narrowTree(
    roots: readonly PageTreeNode[] | null | undefined,
    query: string,
    scope: Scope,
): readonly PageTreeNode[] {
    const needle = query.trim().toLowerCase();
    if (roots === null || roots === undefined) {
        return [];
    }
    if (needle === "" && scope === "all") {
        return roots;
    }
    return roots.map((root) => narrowed(root, needle, scope)).filter((root): root is PageTreeNode => root !== null);
}
