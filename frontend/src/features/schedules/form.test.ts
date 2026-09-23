import { describe, expect, it } from "vitest";

import type { Schedule } from "../../data/types.js";
import {
    cadenceOf,
    createOf,
    dirty,
    draftOf,
    joinInterval,
    maxTargets,
    ready,
    splitInterval,
    updateOf,
} from "./form.js";

function held(overrides: Partial<Schedule> = {}): Schedule {
    return {
        id: "s-1",
        siteId: "site-1",
        name: "Nightly guides",
        cron: "0 3 * * *",
        intervalMinutes: 0,
        entityId: null,
        status: "planned",
        limit: 5,
        templateId: "t-1",
        steps: null,
        publishMode: "draft",
        maxUsd: 4,
        maxTokens: 0,
        enabled: true,
        nextRunAt: "2026-09-22T03:00:00Z",
        lastRunId: null,
        createdBy: "user",
        createdAt: "2026-09-01T00:00:00Z",
        updatedAt: "2026-09-01T00:00:00Z",
        ...overrides,
    };
}

describe("intervals", () => {
    it.each([
        [30, 30, "minutes"],
        [90, 90, "minutes"],
        [360, 6, "hours"],
        [1440, 1, "days"],
        [10080, 7, "days"],
        [0, 1, "days"],
    ] as const)("splits %i minutes into %i %s", (minutes, value, unit) => {
        expect(splitInterval(minutes)).toEqual({ value, unit });
    });

    it("joins back to minutes", () => {
        expect(joinInterval(6, "hours")).toBe(360);
        expect(joinInterval(2, "days")).toBe(2880);
        expect(joinInterval(0, "minutes")).toBe(1);
    });
});

describe("draftOf", () => {
    it("opens a new schedule on a daily interval with nothing named", () => {
        const draft = draftOf(null);
        expect(draft.name).toBe("");
        expect(draft.cadence).toBe("interval");
        expect(draft.limit).toBe(maxTargets);
        expect(draft.enabled).toBe(true);
    });

    it("reads a cron schedule back", () => {
        expect(draftOf(held())).toMatchObject({ cadence: "cron", cron: "0 3 * * *", status: "planned", limit: 5 });
    });

    it("reads an interval schedule back", () => {
        expect(draftOf(held({ cron: "", intervalMinutes: 360 }))).toMatchObject({
            cadence: "interval",
            intervalValue: 6,
            intervalUnit: "hours",
        });
    });
});

describe("cadenceOf sends exactly one cadence", () => {
    it("sends the cron alone", () => {
        expect(cadenceOf({ ...draftOf(held()), cadence: "cron" })).toEqual({ cron: "0 3 * * *", intervalMinutes: 0 });
    });

    it("sends the interval alone", () => {
        const draft = { ...draftOf(held()), cadence: "interval" as const, intervalValue: 30, intervalUnit: "minutes" as const };
        expect(cadenceOf(draft)).toEqual({ cron: "", intervalMinutes: 30 });
    });
});

describe("createOf", () => {
    it("carries the cron and no interval", () => {
        const fields = createOf(draftOf(held()), "site-1");
        expect(fields.cron).toBe("0 3 * * *");
        expect(fields).not.toHaveProperty("intervalMinutes");
        expect(fields.siteId).toBe("site-1");
    });

    it("carries the interval and no cron", () => {
        const fields = createOf(draftOf(held({ cron: "", intervalMinutes: 30 })), "site-1");
        expect(fields.intervalMinutes).toBe(30);
        expect(fields).not.toHaveProperty("cron");
    });

    it("trims the name", () => {
        expect(createOf({ ...draftOf(null), name: "  Weekly  " }, "s").name).toBe("Weekly");
    });
});

describe("updateOf", () => {
    it("sends nothing but the id when nothing moved", () => {
        const current = held();
        expect(updateOf(draftOf(current), current)).toEqual({ id: "s-1" });
        expect(dirty(draftOf(current), current)).toBe(false);
    });

    it("never sends the recipe", () => {
        const current = held({ steps: ["generate_body", "publish"] });
        const draft = { ...draftOf(current), name: "Renamed" };
        expect(updateOf(draft, current)).not.toHaveProperty("steps");
    });

    it("switches a cron schedule to an interval with the interval alone", () => {
        const current = held();
        const draft = { ...draftOf(current), cadence: "interval" as const, intervalValue: 6, intervalUnit: "hours" as const };
        expect(updateOf(draft, current)).toEqual({ id: "s-1", intervalMinutes: 360 });
    });

    it("switches an interval schedule to a cron with the cron alone", () => {
        const current = held({ cron: "", intervalMinutes: 360 });
        const draft = { ...draftOf(current), cadence: "cron" as const, cron: "0 6 * * 1" };
        expect(updateOf(draft, current)).toEqual({ id: "s-1", cron: "0 6 * * 1" });
    });

    it("sends only what changed", () => {
        const current = held();
        const draft = { ...draftOf(current), limit: 20, maxUsd: 9 };
        expect(updateOf(draft, current)).toEqual({ id: "s-1", limit: 20, maxUsd: 9 });
        expect(dirty(draft, current)).toBe(true);
    });

    it("clears a target entity by sending an empty one", () => {
        const current = held({ entityId: "e-1" });
        expect(updateOf({ ...draftOf(current), entityId: "" }, current)).toEqual({ id: "s-1", entityId: "" });
    });
});

describe("ready", () => {
    it("needs a name and a cadence", () => {
        expect(ready(draftOf(null))).toBe(false);
        expect(ready({ ...draftOf(null), name: "Weekly" })).toBe(true);
        expect(ready({ ...draftOf(null), name: "Weekly", cadence: "cron", cron: "  " })).toBe(false);
    });
});
