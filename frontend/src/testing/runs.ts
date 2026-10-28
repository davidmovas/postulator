import type { Run } from "../data/types.js";

export function aRun(overrides: Partial<Run> = {}): Run {
    return {
        id: "r1",
        siteId: "s1",
        kind: "generate",
        status: "running",
        targets: null,
        recipe: null,
        templateId: "t1",
        templateVersion: 1,
        publishMode: "draft",
        budget: { maxUsd: 25, maxTokens: 0 },
        stats: { items: 12, done: 4, failed: 0, tokens: 0, usd: 3.5 },
        createdBy: "user",
        parentRunId: null,
        pauseReason: "",
        error: "",
        deadlineAt: null,
        createdAt: "2026-10-28T09:00:00Z",
        startedAt: "2026-10-28T09:00:01Z",
        finishedAt: null,
        ...overrides,
    };
}

export function aSyncRun(overrides: Partial<Run> = {}): Run {
    return aRun({
        id: "r-sync",
        kind: "sync",
        templateId: "",
        templateVersion: 0,
        targets: ["s1"],
        budget: { maxUsd: 0, maxTokens: 0 },
        stats: { items: 1, done: 0, failed: 0, tokens: 0, usd: 0 },
        ...overrides,
    });
}
