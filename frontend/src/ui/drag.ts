import type { CSSProperties } from "react";

declare module "react" {
    interface CSSProperties {
        "--wails-draggable"?: "drag" | "no-drag";
    }
}

export const dragRegion: CSSProperties = { "--wails-draggable": "drag" };

export const noDrag: CSSProperties = { "--wails-draggable": "no-drag" };
