import type { Palette } from "../../../canvas/palette.js";
import type { Point } from "../../../canvas/viewport.js";
import type { IconComponent, Tone } from "../../../ui/index.js";
import { iconViewBox } from "../../../ui/index.js";

const paths = new Map<string, Path2D>();

function pathOf(icon: IconComponent): Path2D {
    const held = paths.get(icon.path);
    if (held !== undefined) {
        return held;
    }
    const built = new Path2D(icon.path);
    paths.set(icon.path, built);
    return built;
}

export function toneColor(palette: Palette, tone: Tone): string {
    switch (tone) {
        case "accent":
            return palette.accent;
        case "ok":
            return palette.ok;
        case "warn":
            return palette.warn;
        case "danger":
            return palette.danger;
        case "info":
            return palette.info;
        case "muted":
            return palette.inkSoft;
    }
}

export function drawIcon(context: CanvasRenderingContext2D, icon: IconComponent, x: number, y: number, size: number, color: string): void {
    context.save();
    context.translate(x, y + size);
    const scale = size / iconViewBox;
    context.scale(scale, scale);
    context.fillStyle = color;
    context.fill(pathOf(icon));
    context.restore();
}

export function roundRect(context: CanvasRenderingContext2D, x: number, y: number, width: number, height: number, radius: number): void {
    const r = Math.min(radius, width / 2, height / 2);
    context.beginPath();
    context.moveTo(x + r, y);
    context.lineTo(x + width - r, y);
    context.arcTo(x + width, y, x + width, y + r, r);
    context.lineTo(x + width, y + height - r);
    context.arcTo(x + width, y + height, x + width - r, y + height, r);
    context.lineTo(x + r, y + height);
    context.arcTo(x, y + height, x, y + height - r, r);
    context.lineTo(x, y + r);
    context.arcTo(x, y, x + r, y, r);
    context.closePath();
}

export function connector(context: CanvasRenderingContext2D, from: Point, to: Point, color: string, width: number, dash: readonly number[]): void {
    const reach = Math.max((to.x - from.x) / 2, 16);
    context.beginPath();
    context.moveTo(from.x, from.y);
    context.bezierCurveTo(from.x + reach, from.y, to.x - reach, to.y, to.x, to.y);
    context.strokeStyle = color;
    context.lineWidth = width;
    context.setLineDash(dash as number[]);
    context.stroke();
    context.setLineDash([]);
}

export function arc(context: CanvasRenderingContext2D, from: Point, to: Point, color: string, width: number, dash: readonly number[]): void {
    const dx = to.x - from.x;
    const dy = to.y - from.y;
    const distance = Math.hypot(dx, dy) || 1;
    const bulge = Math.min(distance * 0.25, 120);
    const control = { x: (from.x + to.x) / 2 - (dy / distance) * bulge, y: (from.y + to.y) / 2 + (dx / distance) * bulge };
    context.beginPath();
    context.moveTo(from.x, from.y);
    context.quadraticCurveTo(control.x, control.y, to.x, to.y);
    context.strokeStyle = color;
    context.lineWidth = width;
    context.setLineDash(dash as number[]);
    context.stroke();
    context.setLineDash([]);
}

export function chevron(context: CanvasRenderingContext2D, x: number, y: number, size: number, open: boolean, color: string): void {
    context.save();
    context.translate(x + size / 2, y + size / 2);
    if (open) {
        context.rotate(Math.PI / 2);
    }
    context.beginPath();
    context.moveTo(-size / 4, -size / 2 + 1);
    context.lineTo(size / 4, 0);
    context.lineTo(-size / 4, size / 2 - 1);
    context.strokeStyle = color;
    context.lineWidth = 1.5;
    context.lineJoin = "round";
    context.lineCap = "round";
    context.stroke();
    context.restore();
}
