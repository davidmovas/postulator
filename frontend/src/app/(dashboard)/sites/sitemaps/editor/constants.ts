import { BackgroundVariant } from "@xyflow/react";

export const EDITOR = {
    TIMING: {
        HOVER_DELAY_MS: 200,
        MODE_SWITCH_DELAY_MS: 50,
        DIALOG_CLOSE_DELAY_MS: 100,
    },
    FIT_VIEW: {
        PADDING: 0.3,
        MAX_ZOOM: 0.8,
        GO_TO_NODE_PADDING: 0.5,
        GO_TO_NODE_MAX_ZOOM: 1,
        GO_TO_NODE_DURATION: 300,
    },
    CANVAS: {
        MIN_ZOOM: 0.1,
        MAX_ZOOM: 1.5,
        DEFAULT_ZOOM: 0.7,
        SNAP_GRID: [15, 15] as [number, number],
        BACKGROUND_VARIANT: BackgroundVariant.Dots,
        BACKGROUND_GAP: 12,
        BACKGROUND_SIZE: 1,
    },
    HIGHLIGHTING: {
        INCOMING_COLOR: "#22d3ee",
        OUTGOING_COLOR: "#34d399",
        HIERARCHY_COLOR: "#9ca3af",
        HIGHLIGHT_WIDTH: 3,
    },
} as const;
