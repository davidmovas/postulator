import type { Layer } from "./patch.js";

export type ConflictKind = "none" | "template" | "override";

export interface Stamp {
    version: number;
    updatedAt: string;
    overrideUpdatedAt: string | null;
}

export function stampsAgree(opened: Stamp, current: Stamp): boolean {
    return (
        opened.version === current.version &&
        opened.updatedAt === current.updatedAt &&
        opened.overrideUpdatedAt === current.overrideUpdatedAt
    );
}

export function conflictOf(layer: Layer, opened: Stamp | null, current: Stamp, dirty: boolean): ConflictKind {
    if (!dirty || opened === null) {
        return "none";
    }
    if (opened.version !== current.version || opened.updatedAt !== current.updatedAt) {
        return "template";
    }
    if (layer !== "global" && opened.overrideUpdatedAt !== current.overrideUpdatedAt) {
        return "override";
    }
    return "none";
}
