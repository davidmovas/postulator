export interface Point {
    x: number;
    y: number;
}

export interface Size {
    width: number;
    height: number;
}

export interface Rect {
    x: number;
    y: number;
    width: number;
    height: number;
}

export interface Viewport {
    x: number;
    y: number;
    k: number;
}

export const minZoom = 0.1;
export const maxZoom = 3;

export const identity: Viewport = { x: 0, y: 0, k: 1 };

function clampScale(k: number): number {
    return Math.min(Math.max(k, minZoom), maxZoom);
}

export function toScreen(view: Viewport, world: Point): Point {
    return { x: view.x + world.x * view.k, y: view.y + world.y * view.k };
}

export function toWorld(view: Viewport, screen: Point): Point {
    return { x: (screen.x - view.x) / view.k, y: (screen.y - view.y) / view.k };
}

export function zoomAt(view: Viewport, anchor: Point, factor: number): Viewport {
    const k = clampScale(view.k * factor);
    if (k === view.k) {
        return view;
    }
    const ratio = k / view.k;
    return {
        x: anchor.x - (anchor.x - view.x) * ratio,
        y: anchor.y - (anchor.y - view.y) * ratio,
        k,
    };
}

export function panBy(view: Viewport, dx: number, dy: number): Viewport {
    return { x: view.x + dx, y: view.y + dy, k: view.k };
}

export function centerOn(view: Viewport, world: Point, size: Size): Viewport {
    return {
        x: size.width / 2 - world.x * view.k,
        y: size.height / 2 - world.y * view.k,
        k: view.k,
    };
}

export function fit(bounds: Rect, size: Size, padding: number): Viewport {
    const room = { width: Math.max(size.width - padding * 2, 1), height: Math.max(size.height - padding * 2, 1) };
    const scale =
        bounds.width <= 0 || bounds.height <= 0
            ? 1
            : Math.min(room.width / bounds.width, room.height / bounds.height, 1);
    const k = clampScale(scale);
    const centre = { x: bounds.x + bounds.width / 2, y: bounds.y + bounds.height / 2 };
    return centerOn({ x: 0, y: 0, k }, centre, size);
}

export function visibleWorld(view: Viewport, size: Size): Rect {
    const origin = toWorld(view, { x: 0, y: 0 });
    return { x: origin.x, y: origin.y, width: size.width / view.k, height: size.height / view.k };
}
