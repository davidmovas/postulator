import type { ColumnLayout, Tracks } from "../../../ui/collapse.js";
import { laidOut, widestFitting } from "../../../ui/collapse.js";

const rowChrome = 24;

export type PurposeColumn = "purpose" | "calls" | "failed" | "spent" | "share";

const purposeTracks: Tracks<PurposeColumn> = {
    purpose: { track: "minmax(8rem,1.4fr)", least: 128 },
    calls: { track: "4.5rem", least: 72 },
    failed: { track: "4.5rem", least: 72 },
    spent: { track: "5.5rem", least: 88 },
    share: { track: "minmax(7rem,1fr)", least: 112 },
};

const purposeLayouts: readonly ColumnLayout<PurposeColumn>[] = [
    laidOut(["purpose", "calls", "failed", "spent", "share"], purposeTracks, rowChrome),
    laidOut(["purpose", "calls", "spent", "share"], purposeTracks, rowChrome),
    laidOut(["purpose", "calls", "spent"], purposeTracks, rowChrome),
];

const purposeNarrowest = laidOut<PurposeColumn>(
    ["purpose", "calls", "spent"],
    { ...purposeTracks, purpose: { track: "minmax(0,1fr)", least: 0 } },
    rowChrome,
);

export function purposeLayout(width: number): ColumnLayout<PurposeColumn> {
    return widestFitting(width, purposeLayouts, purposeNarrowest);
}

export type ModelColumn = "model" | "tier" | "calls" | "input" | "cached" | "output" | "reasoning" | "spent";

const modelTracks: Tracks<ModelColumn> = {
    model: { track: "minmax(9rem,1.6fr)", least: 144 },
    tier: { track: "5rem", least: 80 },
    calls: { track: "4rem", least: 64 },
    input: { track: "4.5rem", least: 72 },
    cached: { track: "4.5rem", least: 72 },
    output: { track: "4.5rem", least: 72 },
    reasoning: { track: "4.5rem", least: 72 },
    spent: { track: "5rem", least: 80 },
};

const modelLayouts: readonly ColumnLayout<ModelColumn>[] = [
    laidOut(["model", "tier", "calls", "input", "cached", "output", "reasoning", "spent"], modelTracks, rowChrome),
    laidOut(["model", "tier", "calls", "input", "output", "spent"], modelTracks, rowChrome),
    laidOut(["model", "calls", "input", "output", "spent"], modelTracks, rowChrome),
    laidOut(["model", "calls", "spent"], modelTracks, rowChrome),
];

const modelNarrowest = laidOut<ModelColumn>(
    ["model", "calls", "spent"],
    { ...modelTracks, model: { track: "minmax(0,1fr)", least: 0 } },
    rowChrome,
);

export function modelLayout(width: number): ColumnLayout<ModelColumn> {
    return widestFitting(width, modelLayouts, modelNarrowest);
}

export type CallColumn = "when" | "what" | "model" | "tier" | "tokens" | "spent";

const callTracks: Tracks<CallColumn> = {
    when: { track: "6.5rem", least: 104 },
    what: { track: "minmax(9rem,1.4fr)", least: 144 },
    model: { track: "minmax(8rem,1fr)", least: 128 },
    tier: { track: "5rem", least: 80 },
    tokens: { track: "9rem", least: 144 },
    spent: { track: "4.5rem", least: 72 },
};

const callLayouts: readonly ColumnLayout<CallColumn>[] = [
    laidOut(["when", "what", "model", "tier", "tokens", "spent"], callTracks, rowChrome),
    laidOut(["when", "what", "model", "tokens", "spent"], callTracks, rowChrome),
    laidOut(["when", "what", "model", "spent"], callTracks, rowChrome),
    laidOut(["what", "model", "spent"], callTracks, rowChrome),
];

const callNarrowest = laidOut<CallColumn>(
    ["what", "model", "spent"],
    {
        ...callTracks,
        what: { track: "minmax(0,1.4fr)", least: 0 },
        model: { track: "minmax(0,1fr)", least: 0 },
    },
    rowChrome,
);

export function callLayout(width: number): ColumnLayout<CallColumn> {
    return widestFitting(width, callLayouts, callNarrowest);
}
