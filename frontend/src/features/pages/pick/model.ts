import type { Page, PageTreeNode } from "../../../data/types.js";

export interface PickIndex {
    byId: ReadonlyMap<string, Page>;
    parentOf: ReadonlyMap<string, string>;
    childrenOf: ReadonlyMap<string, readonly string[]>;
}

export type Tick = "on" | "off" | "some";

export type Keep = (page: Page) => boolean;

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

export function branchOf(id: string, index: PickIndex, pickable: Keep): string[] {
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

export function toggled(held: ReadonlySet<string>, ids: readonly string[], on: boolean): ReadonlySet<string> {
    const next = new Set(held);
    for (const id of ids) {
        if (on) {
            next.add(id);
        } else {
            next.delete(id);
        }
    }
    return next;
}

function matches(page: Page, query: string, keep: Keep): boolean {
    if (!keep(page)) {
        return false;
    }
    return query === "" || page.path.toLowerCase().includes(query) || page.title.toLowerCase().includes(query);
}

function narrowed(node: PageTreeNode, query: string, keep: Keep): PageTreeNode | null {
    const kept = childrenOf(node)
        .map((child) => narrowed(child, query, keep))
        .filter((child): child is PageTreeNode => child !== null);
    if (kept.length > 0 || matches(node.page, query, keep)) {
        return { page: node.page, children: kept };
    }
    return null;
}

export const everyPage: Keep = () => true;

export function narrowTree(
    roots: readonly PageTreeNode[] | null | undefined,
    query: string,
    keep: Keep,
): readonly PageTreeNode[] {
    const needle = query.trim().toLowerCase();
    if (roots === null || roots === undefined) {
        return [];
    }
    if (needle === "" && keep === everyPage) {
        return roots;
    }
    return roots.map((root) => narrowed(root, needle, keep)).filter((root): root is PageTreeNode => root !== null);
}
