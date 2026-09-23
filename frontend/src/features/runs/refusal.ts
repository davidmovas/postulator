import type { Page, RunItem } from "../../data/types.js";
import { itemPaused, pauseNeedsHuman, stepPublish } from "./statuses.js";

export interface DriftRefusal {
    path: string;
    lastSyncedAt: string | null;
    wpModifiedAt: string | null;
}

export function driftRefusal(item: RunItem | null, page: Page | undefined): DriftRefusal | null {
    if (item === null || page === undefined || !page.drift) {
        return null;
    }
    if (item.status !== itemPaused || item.pauseReason !== pauseNeedsHuman) {
        return null;
    }
    if (item.currentStep !== stepPublish) {
        return null;
    }
    return { path: page.path, lastSyncedAt: page.lastSyncedAt, wpModifiedAt: page.wpModifiedAt };
}
