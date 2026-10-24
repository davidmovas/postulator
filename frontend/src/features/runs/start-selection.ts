import type { AddedPage, SiteMapNode } from "../../data/types.js";
import type { Keep } from "../pages/pick/model.js";
import { everyNode } from "../pages/pick/model.js";

export type Scope = "all" | "missing" | "present";

export const scopes: readonly Scope[] = ["all", "missing", "present"];

const keySeparator = "\n";

export function pickable(node: SiteMapNode): boolean {
    return (node.entityId ?? "") !== "";
}

const notOnSite: Keep = (node) => !node.page.onSite;

const onSite: Keep = (node) => node.page.onSite;

export function inScope(scope: Scope): Keep {
    switch (scope) {
        case "missing":
            return notOnSite;
        case "present":
            return onSite;
        default:
            return everyNode;
    }
}

export function requiredOf(added: readonly AddedPage[] | null | undefined, selected: ReadonlySet<string>): ReadonlyMap<string, string> {
    const required = new Map<string, string>();
    if (selected.size === 0) {
        return required;
    }
    for (const parent of added ?? []) {
        if (!selected.has(parent.pageId) && !required.has(parent.pageId)) {
            required.set(parent.pageId, parent.neededBy);
        }
    }
    return required;
}

export function selectionKey(selected: ReadonlySet<string>): string {
    return [...selected].sort().join(keySeparator);
}

export function idsOf(key: string): string[] {
    return key === "" ? [] : key.split(keySeparator);
}
