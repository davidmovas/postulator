import type { SyncProgress } from "../data/types.js";

export function aProgress(overrides: Partial<SyncProgress> = {}): SyncProgress {
    return {
        found: true,
        runId: "r-sync",
        status: "running",
        stage: "render",
        source: "plugin",
        full: false,
        batches: 6,
        pulled: 150,
        created: 3,
        updated: 12,
        products: 40,
        rendered: 120,
        unreachable: 2,
        archived: 0,
        entities: 0,
        placed: 0,
        unplaced: 0,
        unplacedBy: {},
        attempts: 0,
        error: "",
        wakeAt: null,
        startedAt: "2026-10-28T09:00:00Z",
        createdAt: "2026-10-28T08:59:58Z",
        ...overrides,
    };
}

export const noSync: SyncProgress = aProgress({
    found: false,
    runId: "",
    status: "",
    stage: "",
    source: "",
    batches: 0,
    pulled: 0,
    created: 0,
    updated: 0,
    products: 0,
    rendered: 0,
    unreachable: 0,
    startedAt: null,
});
