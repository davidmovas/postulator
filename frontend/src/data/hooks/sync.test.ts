import { describe, expect, it } from "vitest";

import { aProgress, noSync } from "../../testing/sync.js";
import { keys } from "../keys.js";
import { syncPoll, syncPollMs, syncUnderWay } from "./sync.js";

describe("a site's sync under way", () => {
    it.each(["pending", "running", "waiting", "paused"])("is under way while its run is %s", (status) => {
        expect(syncUnderWay(aProgress({ status }))).toBe(true);
    });

    it.each(["completed", "failed", "cancelled"])("is over once its run is %s", (status) => {
        expect(syncUnderWay(aProgress({ status }))).toBe(false);
    });

    it("is no sync at all when the site has none, or before the answer arrives", () => {
        expect(syncUnderWay(noSync)).toBe(false);
        expect(syncUnderWay(undefined)).toBe(false);
    });
});

describe("how often a sync's progress is read without an event", () => {
    it("asks again on a modest interval while the sync goes on", () => {
        expect(syncPoll(aProgress({ status: "waiting" }))).toBe(syncPollMs);
        expect(syncPollMs).toBeGreaterThanOrEqual(2000);
    });

    it("stops asking once the sync is over or when there is none", () => {
        expect(syncPoll(aProgress({ status: "completed" }))).toBe(false);
        expect(syncPoll(noSync)).toBe(false);
        expect(syncPoll(undefined)).toBe(false);
    });
});

describe("the keys a sync's progress is held under", () => {
    it("keeps a site's sync under way apart from one named run of it", () => {
        expect(keys.sync.progress("s1", "")).not.toEqual(keys.sync.progress("s1", "r1"));
        expect(keys.sync.progress("s1", "r1").slice(0, keys.sync.progresses().length)).toEqual([
            ...keys.sync.progresses(),
        ]);
    });
});
