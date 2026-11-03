import type { AddressNode, SiteMapNode } from "../../../data/types.js";
import type { ChildrenOf, SelectionRow } from "../../../ui/selection.js";
import type { TypeIdOf } from "../views/model.js";

export const rootFolder = "/";

export interface AddressBranch {
    nodes: readonly AddressNode[];
    hasMore: boolean;
    pending: boolean;
}

export type AddressBranches = ReadonlyMap<string, AddressBranch>;

export type AddressRow =
    | { kind: "node"; key: string; node: AddressNode; depth: number; expanded: boolean; ancestors: readonly string[] }
    | { kind: "more"; key: string; folder: string; depth: number }
    | { kind: "loading"; key: string; folder: string; depth: number };

export function keyOf(node: AddressNode): string {
    return node.page === undefined || node.page === null ? node.path : node.page.page.pageId;
}

function opens(node: AddressNode, expanded: ReadonlySet<string>): boolean {
    return node.children > 0 && expanded.has(node.path);
}

export function openFolders(expanded: ReadonlySet<string>, branches: AddressBranches): string[] {
    const out = [""];
    const seen = new Set(out);
    const walk = (folder: string): void => {
        for (const node of branches.get(folder)?.nodes ?? []) {
            if (opens(node, expanded) && !seen.has(node.path)) {
                seen.add(node.path);
                out.push(node.path);
                walk(node.path);
            }
        }
    };
    walk("");
    return out;
}

export function flattenAddress(expanded: ReadonlySet<string>, branches: AddressBranches): AddressRow[] {
    const out: AddressRow[] = [];
    const seen = new Set([""]);
    const walk = (folder: string, depth: number, ancestors: readonly string[]): void => {
        const branch = branches.get(folder);
        if (branch === undefined || (branch.pending && branch.nodes.length === 0)) {
            out.push({ kind: "loading", key: `loading:${folder}`, folder, depth });
            return;
        }
        for (const node of branch.nodes) {
            const key = keyOf(node);
            const open = opens(node, expanded) && !seen.has(node.path);
            out.push({ kind: "node", key: `node:${node.path}`, node, depth, expanded: open, ancestors });
            if (open) {
                seen.add(node.path);
                walk(node.path, depth + 1, [key, ...ancestors]);
            }
        }
        if (branch.hasMore) {
            out.push({ kind: "more", key: `more:${folder}`, folder, depth });
        }
    };
    walk("", 0, []);
    return out;
}

export function addressRow(node: AddressNode, ancestors: readonly string[], typeIdOf: TypeIdOf): SelectionRow {
    const type = node.page?.page.type ?? "";
    return { ancestors, typeId: type === "" ? null : typeIdOf(type) };
}

export function asMapNode(node: AddressNode): SiteMapNode | null {
    if (node.page === undefined || node.page === null) {
        return null;
    }
    return { ...node.page, children: node.children, descendants: node.pages - 1 };
}

export function addressChildren(
    branches: AddressBranches,
    folderOf: (key: string) => string | undefined,
    chainOf: (key: string) => readonly string[] | undefined,
    typeIdOf: TypeIdOf,
    within: string | undefined,
): ChildrenOf {
    return (key) => {
        const folder = folderOf(key);
        const chain = chainOf(key);
        const branch = folder === undefined ? undefined : branches.get(folder);
        if (chain === undefined || branch === undefined || branch.pending || branch.hasMore) {
            return null;
        }
        const ancestors = [key, ...chain];
        return branch.nodes.map((node) => ({
            id: keyOf(node),
            row: addressRow(node, ancestors, typeIdOf),
            size: within === undefined ? node.pages : null,
        }));
    };
}
