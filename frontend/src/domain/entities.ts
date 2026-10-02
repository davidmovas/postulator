import type { Entity } from "../data/types.js";

const separator = " › ";

function nameKey(name: string): string {
    return name.trim().toLowerCase();
}

export function entityLabels(entities: readonly Entity[]): ReadonlyMap<string, string> {
    const byId = new Map<string, Entity>();
    const uses = new Map<string, number>();
    for (const held of entities) {
        byId.set(held.id, held);
        uses.set(nameKey(held.name), (uses.get(nameKey(held.name)) ?? 0) + 1);
    }
    const shared = (name: string): boolean => (uses.get(nameKey(name)) ?? 0) > 1;

    const labelOf = (held: Entity, climbed: Set<string>): string => {
        if (held.scopeEntityId === null || !shared(held.name)) {
            return held.name;
        }
        const parent = byId.get(held.scopeEntityId);
        if (parent === undefined || climbed.has(held.id)) {
            return held.name;
        }
        climbed.add(held.id);
        const above = shared(parent.name) ? labelOf(parent, climbed) : parent.name;
        return above + separator + held.name;
    };

    const labels = new Map<string, string>();
    for (const held of entities) {
        labels.set(held.id, labelOf(held, new Set()));
    }
    return labels;
}
