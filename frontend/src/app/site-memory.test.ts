import { describe, expect, it } from "vitest";

import { landing } from "./site-memory.js";

describe("landing", () => {
    const cases: readonly { name: string; last: string | null; ids: string[]; want: string }[] = [
        { name: "no sites goes to the sites screen", last: null, ids: [], want: "/sites" },
        {
            name: "a remembered site that is gone falls back to the first one",
            last: "deleted",
            ids: ["s1", "s2"],
            want: "/s/s1/overview",
        },
        {
            name: "a remembered site wins when it still exists",
            last: "s2",
            ids: ["s1", "s2"],
            want: "/s/s2/overview",
        },
        {
            name: "nothing remembered opens the first site",
            last: null,
            ids: ["s1", "s2"],
            want: "/s/s1/overview",
        },
        { name: "a remembered site is ignored when nothing is listed", last: "s1", ids: [], want: "/sites" },
    ];

    it.each(cases)("$name", ({ last, ids, want }) => {
        expect(landing(last, ids)).toBe(want);
    });
});
