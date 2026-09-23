import type { ReactElement } from "react";
import { useCallback, useEffect, useRef } from "react";

import { CanvasHost } from "../../../canvas/host.js";
import type { CanvasHandle, PointerInfo } from "../../../canvas/host.js";
import type { Palette } from "../../../canvas/palette.js";
import type { Layout } from "../../../canvas/tree-layout.js";
import { visibleWorld } from "../../../canvas/viewport.js";
import type { Point, Size } from "../../../canvas/viewport.js";
import { copy } from "../../../copy/index.js";
import { useViewport } from "../state.js";

const width = 168;
const height = 104;
const inset = 6;

export interface MinimapProps {
    siteId: string;
    layout: Layout;
    palette: Palette;
    mapSize: () => Size;
    onJump: (world: Point) => void;
}

export function Minimap({ siteId, layout, palette, mapSize, onJump }: MinimapProps): ReactElement {
    const host = useRef<CanvasHandle>(null);
    const view = useViewport(siteId);
    const latest = useRef({ layout, palette, view, mapSize });
    latest.current = { layout, palette, view, mapSize };

    const scaleOf = useCallback((): { k: number; ox: number; oy: number } => {
        const bounds = latest.current.layout.bounds;
        const k = Math.min((width - inset * 2) / Math.max(bounds.width, 1), (height - inset * 2) / Math.max(bounds.height, 1));
        return { k, ox: inset - bounds.x * k, oy: inset - bounds.y * k };
    }, []);

    useEffect(() => {
        host.current?.redraw();
    }, [layout, view, palette]);

    const draw = useCallback(
        (context: CanvasRenderingContext2D): void => {
            const { layout: placed, palette: colors, view: current, mapSize: size } = latest.current;
            const { k, ox, oy } = scaleOf();
            context.fillStyle = colors.inkFaint;
            for (const node of placed.nodes) {
                context.fillRect(ox + node.x * k, oy + node.y * k, Math.max(node.width * k, 2), Math.max(node.height * k, 1.5));
            }
            if (current !== null) {
                const seen = visibleWorld(current, size());
                context.strokeStyle = colors.accent;
                context.lineWidth = 1;
                context.strokeRect(ox + seen.x * k, oy + seen.y * k, seen.width * k, seen.height * k);
                context.fillStyle = colors.accentSoft;
                context.fillRect(ox + seen.x * k, oy + seen.y * k, seen.width * k, seen.height * k);
            }
        },
        [scaleOf],
    );

    const jump = (pointer: PointerInfo): void => {
        const { k, ox, oy } = scaleOf();
        onJump({ x: (pointer.x - ox) / k, y: (pointer.y - oy) / k });
    };

    return (
        <div
            className="absolute right-3 bottom-3 overflow-hidden rounded-md border border-hairline bg-panel/90 backdrop-blur"
            style={{ width, height }}
        >
            <CanvasHost ref={host} label={copy.graph.controls.minimap} draw={draw} onPointerDown={jump} cursor="crosshair" />
        </div>
    );
}
