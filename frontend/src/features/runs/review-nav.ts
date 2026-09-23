export interface Identified {
    id: string;
}

export function neighbourOf(items: readonly Identified[], current: string, delta: number): string | null {
    const at = items.findIndex((held) => held.id === current);
    if (at < 0) {
        return null;
    }
    const next = at + delta;
    if (next < 0 || next >= items.length) {
        return null;
    }
    return items[next].id;
}

export function positionOf(items: readonly Identified[], current: string): number {
    return items.findIndex((held) => held.id === current) + 1;
}
