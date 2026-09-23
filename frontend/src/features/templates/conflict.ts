import type { Layer } from "./patch.js";
import type { SpecDraft } from "./spec.js";
import { specJsonOf } from "./spec.js";

export type ConflictKind = "none" | "template" | "override";

export interface Stamp {
    beneath: string;
    overrideUpdatedAt: string | null;
}

export function beneathOf(draft: SpecDraft | null): string {
    return draft === null ? "" : JSON.stringify(specJsonOf(draft));
}

export function conflictOf(layer: Layer, opened: Stamp | null, current: Stamp, dirty: boolean): ConflictKind {
    if (!dirty || opened === null) {
        return "none";
    }
    if (opened.beneath !== current.beneath) {
        return "template";
    }
    if (layer !== "global" && opened.overrideUpdatedAt !== current.overrideUpdatedAt) {
        return "override";
    }
    return "none";
}
