import type { ColumnLayout, Tracks } from "../../ui/collapse.js";
import { laidOut, widestFitting } from "../../ui/collapse.js";

export type RunColumn = "kind" | "status" | "progress" | "spend" | "started" | "lasted";

export type RunLayout = ColumnLayout<RunColumn>;

const tracks: Tracks<RunColumn> = {
    kind: { track: "64px", least: 64 },
    status: { track: "192px", least: 192 },
    progress: { track: "minmax(150px,1.3fr)", least: 150 },
    spend: { track: "80px", least: 80 },
    started: { track: "96px", least: 96 },
    lasted: { track: "72px", least: 72 },
};

const shrinking: Tracks<RunColumn> = {
    ...tracks,
    status: { track: "minmax(0,1.2fr)", least: 0 },
    progress: { track: "minmax(0,1fr)", least: 0 },
};

const rowChrome = 24;

const layouts: readonly RunLayout[] = [
    laidOut(["kind", "status", "progress", "spend", "started", "lasted"], tracks, rowChrome),
    laidOut(["kind", "status", "progress", "spend", "started"], tracks, rowChrome),
    laidOut(["kind", "status", "progress", "started"], tracks, rowChrome),
    laidOut(["kind", "status", "progress"], tracks, rowChrome),
];

const narrowest = laidOut<RunColumn>(["kind", "status", "progress"], shrinking, rowChrome);

export function runLayout(width: number): RunLayout {
    return widestFitting(width, layouts, narrowest);
}
