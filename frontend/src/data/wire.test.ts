import { describe, expect, test } from "vitest";

import type { PagingModels, RunsModels, SitesModels } from "../lib/api.js";
import type { Timestamp, Wire } from "./wire.js";

type Equal<A, B> = (<T>() => T extends A ? 1 : 2) extends (<T>() => T extends B ? 1 : 2) ? true : false;

describe("Wire narrowing", () => {
    test("narrows every dto.Time field by name", () => {
        const createdAt: Equal<Wire<RunsModels.Run>["createdAt"], Timestamp> = true;
        const finishedAt: Equal<Wire<RunsModels.Run>["finishedAt"], Timestamp> = true;
        const deadlineAt: Equal<Wire<RunsModels.Run>["deadlineAt"], Timestamp> = true;
        const wpModifiedAt: Equal<Wire<SitesModels.Site>["updatedAt"], Timestamp> = true;
        expect([createdAt, finishedAt, deadlineAt, wpModifiedAt]).toEqual([true, true, true, true]);
    });

    test("narrows a replayed event timestamp even though dto.Time generates as any", () => {
        const at: Equal<Wire<RunsModels.Event>["at"], Timestamp> = true;
        expect(at).toBe(true);
    });

    test("leaves RawMessage opaque", () => {
        const notNarrowed: Equal<Wire<RunsModels.Event>["payload"], Timestamp> = false;
        expect(notNarrowed).toBe(false);

        const payload: Wire<RunsModels.Event>["payload"] = { runId: "r", nested: [1, 2, 3] };
        expect(payload).toEqual({ runId: "r", nested: [1, 2, 3] });
    });

    test("leaves Slice opaque", () => {
        const items: Wire<PagingModels.List<RunsModels.Run>>["items"] = [1, "two", null];
        expect(items).toHaveLength(3);
    });

    test("recurses into nested objects without touching non-timestamp fields", () => {
        const budget: Wire<RunsModels.Run>["budget"] = { maxUsd: 5, maxTokens: 0 };
        expect(budget.maxUsd).toBe(5);

        const capabilities: Equal<Wire<SitesModels.Site>["plugin"]["capabilities"], string[] | null> = true;
        expect(capabilities).toBe(true);
    });

    test("accepts a runtime null for a zero timestamp", () => {
        const absent: Wire<RunsModels.Run>["startedAt"] = null;
        const present: Wire<RunsModels.Run>["startedAt"] = "2026-09-19T10:00:00Z";
        expect([absent, present]).toEqual([null, "2026-09-19T10:00:00Z"]);
    });
});
