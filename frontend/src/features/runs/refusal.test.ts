import { describe, expect, it } from "vitest";

import type { Page, RunItem } from "../../data/types.js";
import { driftRefusal } from "./refusal.js";

const page = {
    id: "p1",
    path: "/collections/mugs/",
    drift: true,
    lastSyncedAt: "2026-09-03T14:22:00Z",
    wpModifiedAt: "2026-09-12T09:41:00Z",
} as unknown as Page;

const item = {
    id: "i1",
    status: "paused",
    pauseReason: "needs_human",
    currentStep: "publish",
} as unknown as RunItem;

describe("driftRefusal", () => {
    it("recognises a publish step that stopped for a human on a drifted page", () => {
        expect(driftRefusal(item, page)).toStrictEqual({
            path: "/collections/mugs/",
            lastSyncedAt: "2026-09-03T14:22:00Z",
            wpModifiedAt: "2026-09-12T09:41:00Z",
        });
    });

    it("is nothing without every part of the refusal", () => {
        expect(driftRefusal(null, page)).toBeNull();
        expect(driftRefusal(item, undefined)).toBeNull();
        expect(driftRefusal(item, { ...page, drift: false })).toBeNull();
        expect(driftRefusal({ ...item, status: "failed" } as RunItem, page)).toBeNull();
        expect(driftRefusal({ ...item, pauseReason: "user" } as RunItem, page)).toBeNull();
        expect(driftRefusal({ ...item, currentStep: "judge" } as RunItem, page)).toBeNull();
    });
});
