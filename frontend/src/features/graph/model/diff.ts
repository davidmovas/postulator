import type { Entity } from "../../../data/types.js";
import type { GraphIndex } from "./index.js";

export interface IndexDiff {
    added: string[];
    removed: string[];
    changedEdges: string[];
    touched: string[];
}

function changed(before: Entity, after: Entity): boolean {
    return (
        before.name !== after.name ||
        before.kind !== after.kind ||
        before.score !== after.score ||
        before.canonicalPageId !== after.canonicalPageId ||
        before.primaryKeyword !== after.primaryKeyword
    );
}

export function diffIndex(before: GraphIndex, after: GraphIndex): IndexDiff {
    const added: string[] = [];
    const removed: string[] = [];
    const changedEdges: string[] = [];
    const touched = new Set<string>();

    for (const held of after.entities) {
        const previous = before.byId.get(held.id);
        if (previous === undefined) {
            added.push(held.id);
            touched.add(held.id);
        } else if (changed(previous, held)) {
            touched.add(held.id);
        }
    }
    for (const id of before.byId.keys()) {
        if (!after.byId.has(id)) {
            removed.push(id);
        }
    }
    for (const [id, edge] of after.edgeById) {
        const previous = before.edgeById.get(id);
        if (previous === undefined || previous.status !== edge.status) {
            changedEdges.push(id);
            touched.add(edge.fromEntityId);
            touched.add(edge.toEntityId);
        }
    }

    const ordered = after.entities.filter((held) => touched.has(held.id)).map((held) => held.id);
    return { added, removed, changedEdges, touched: ordered };
}
