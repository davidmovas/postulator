import type { SiteMapNode } from "../../../data/types.js";
import { between } from "../../../ui/selection.js";
import type { Branches } from "../../map/model/tree.js";

export type Tick = "on" | "off" | "some";

export type Keep = (node: SiteMapNode) => boolean;

export interface NamedPage {
    id: string;
    path: string;
}

export const everyNode: Keep = () => true;

export function tick(id: string, selected: ReadonlySet<string>, required: ReadonlyMap<string, string>, branches: Branches): Tick {
    if (selected.has(id) || required.has(id)) {
        return "on";
    }
    const seen = new Set([id]);
    const stack = [...(branches.get(id)?.nodes ?? [])];
    while (stack.length > 0) {
        const next = stack.pop();
        if (next === undefined) {
            break;
        }
        const below = next.page.pageId;
        if (seen.has(below)) {
            continue;
        }
        seen.add(below);
        if (selected.has(below)) {
            return "some";
        }
        stack.push(...(branches.get(below)?.nodes ?? []));
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

export interface NextPick {
    selected: ReadonlySet<string>;
    named: readonly NamedPage[];
    anchor: string;
}

export function nextPick(
    selected: ReadonlySet<string>,
    shown: readonly NamedPage[],
    anchor: string | null,
    target: NamedPage,
    ranged: boolean,
): NextPick {
    const span =
        ranged && anchor !== null
            ? between(
                  shown.map((page) => page.id),
                  anchor,
                  target.id,
              )
            : [];
    if (anchor === null || span.length === 0) {
        return { selected: toggled(selected, [target.id], !selected.has(target.id)), named: [target], anchor: target.id };
    }
    const spanned = new Set(span);
    return { selected: toggled(selected, span, selected.has(anchor)), named: shown.filter((page) => spanned.has(page.id)), anchor };
}

export function kept(node: SiteMapNode, keep: Keep): boolean {
    return keep(node) || node.children > 0;
}

export function labelOf(node: SiteMapNode, found: boolean): string {
    const path = node.page.path;
    if (found) {
        return path;
    }
    const parts = path.split("/").filter((part) => part !== "");
    return parts.length === 0 ? "/" : `${parts[parts.length - 1]}/`;
}
