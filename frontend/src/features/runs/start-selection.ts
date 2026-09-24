import type { Page, PageTreeNode } from "../../data/types.js";
import type { PickIndex, Tick } from "../pages/pick/model.js";
import { branchOf as branchBelow, everyPage, indexTree, narrowTree as narrowBy, tick, toggled } from "../pages/pick/model.js";

export type { PickIndex, Tick };
export { indexTree, tick, toggled };

export type Scope = "all" | "missing" | "present";

export const scopes: readonly Scope[] = ["all", "missing", "present"];

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
    return branchBelow(id, index, pickable);
}

export function inScope(scope: Scope): (page: Page) => boolean {
    switch (scope) {
        case "missing":
            return (page) => !onSite(page);
        case "present":
            return onSite;
        default:
            return everyPage;
    }
}

export function narrowTree(
    roots: readonly PageTreeNode[] | null | undefined,
    query: string,
    scope: Scope,
): readonly PageTreeNode[] {
    return narrowBy(roots, query, inScope(scope));
}
