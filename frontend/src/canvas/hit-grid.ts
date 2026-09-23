import type { Rect } from "./viewport.js";

export interface HitItem extends Rect {
    id: string;
}

function contains(item: Rect, x: number, y: number): boolean {
    return x >= item.x && x < item.x + item.width && y >= item.y && y < item.y + item.height;
}

function intersects(item: Rect, area: Rect): boolean {
    return (
        item.x < area.x + area.width &&
        item.x + item.width > area.x &&
        item.y < area.y + area.height &&
        item.y + item.height > area.y
    );
}

export class HitGrid<T extends HitItem = HitItem> {
    private readonly cells = new Map<string, T[]>();
    private readonly order = new Map<string, number>();

    constructor(
        items: readonly T[],
        private readonly cell: number,
    ) {
        items.forEach((item, index) => {
            this.order.set(item.id, index);
            this.eachCell(item, (key) => {
                const held = this.cells.get(key);
                if (held === undefined) {
                    this.cells.set(key, [item]);
                } else {
                    held.push(item);
                }
            });
        });
    }

    at(x: number, y: number): T | null {
        const held = this.cells.get(this.key(Math.floor(x / this.cell), Math.floor(y / this.cell)));
        if (held === undefined) {
            return null;
        }
        for (let index = held.length - 1; index >= 0; index -= 1) {
            if (contains(held[index], x, y)) {
                return held[index];
            }
        }
        return null;
    }

    within(area: Rect): T[] {
        const seen = new Set<string>();
        const found: T[] = [];
        this.eachCell(area, (key) => {
            for (const item of this.cells.get(key) ?? []) {
                if (!seen.has(item.id) && intersects(item, area)) {
                    seen.add(item.id);
                    found.push(item);
                }
            }
        });
        found.sort((left, right) => (this.order.get(left.id) ?? 0) - (this.order.get(right.id) ?? 0));
        return found;
    }

    private key(column: number, row: number): string {
        return `${column}:${row}`;
    }

    private eachCell(area: Rect, visit: (key: string) => void): void {
        if (area.width <= 0 || area.height <= 0) {
            return;
        }
        const firstColumn = Math.floor(area.x / this.cell);
        const lastColumn = Math.floor((area.x + area.width - Number.EPSILON) / this.cell);
        const firstRow = Math.floor(area.y / this.cell);
        const lastRow = Math.floor((area.y + area.height - Number.EPSILON) / this.cell);
        for (let column = firstColumn; column <= lastColumn; column += 1) {
            for (let row = firstRow; row <= lastRow; row += 1) {
                visit(this.key(column, row));
            }
        }
    }
}
