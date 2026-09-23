import type { VisibleEntity, VisibleRow } from "./fold.js";

export type NavKey = "up" | "down" | "left" | "right" | "home" | "end";

export interface Move {
    id: string;
    expand: boolean;
}

function step(id: string): Move {
    return { id, expand: false };
}

export function move(rows: readonly VisibleRow[], selectedId: string | null, key: NavKey): Move | null {
    const entities = rows.filter((row): row is VisibleEntity => row.kind === "entity");
    if (entities.length === 0) {
        return null;
    }
    const at = selectedId === null ? -1 : entities.findIndex((row) => row.id === selectedId);
    if (selectedId !== null && at === -1) {
        return null;
    }
    switch (key) {
        case "home":
            return step(entities[0].id);
        case "end":
            return step(entities[entities.length - 1].id);
        case "down":
            if (at === -1) {
                return step(entities[0].id);
            }
            return at + 1 < entities.length ? step(entities[at + 1].id) : null;
        case "up":
            if (at === -1) {
                return step(entities[0].id);
            }
            return at > 0 ? step(entities[at - 1].id) : null;
        case "left": {
            if (at === -1) {
                return null;
            }
            const parentId = entities[at].parentId;
            return parentId === null ? null : step(parentId);
        }
        case "right": {
            if (at === -1) {
                return null;
            }
            const row = entities[at];
            if (row.childCount === 0) {
                return null;
            }
            if (!row.expanded) {
                return { id: row.id, expand: true };
            }
            const child = entities.find((held, index) => index > at && held.parentId === row.id);
            return child === undefined ? null : step(child.id);
        }
    }
}
