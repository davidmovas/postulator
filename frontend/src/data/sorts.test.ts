import { describe, expect, it } from "vitest";

import { cycleSort, formatSort, parseSort, sortSegment } from "./sorts.js";

const fields = ["path", "createdAt"] as const;

describe("parseSort", () => {
    const cases: readonly { name: string; raw: string | null; want: { field: string; desc: boolean } | null }[] = [
        { name: "an ascending field", raw: "path:asc", want: { field: "path", desc: false } },
        { name: "a descending field", raw: "createdAt:desc", want: { field: "createdAt", desc: true } },
        { name: "a field the list does not declare", raw: "status:asc", want: null },
        { name: "a direction that is neither", raw: "path:sideways", want: null },
        { name: "a field with no direction", raw: "path", want: null },
        { name: "a direction with no field", raw: ":asc", want: null },
        { name: "an empty value", raw: "", want: null },
        { name: "no value", raw: null, want: null },
    ];

    for (const held of cases) {
        it(held.name, () => {
            expect(parseSort(fields, held.raw)).toStrictEqual(held.want);
        });
    }
});

describe("formatSort", () => {
    it("writes the field and its direction, and nothing for no sort", () => {
        expect(formatSort({ field: "path", desc: true })).toBe("path:desc");
        expect(formatSort({ field: "createdAt", desc: false })).toBe("createdAt:asc");
        expect(formatSort(null)).toBe("");
        expect(formatSort(undefined)).toBe("");
    });

    it("is read back by parseSort", () => {
        for (const sort of [
            { field: "path" as const, desc: false },
            { field: "createdAt" as const, desc: true },
        ]) {
            expect(parseSort(fields, formatSort(sort))).toStrictEqual(sort);
        }
    });
});

describe("sortSegment", () => {
    it("names the backend's own order when there is no sort", () => {
        expect(sortSegment(null)).toBe("default");
        expect(sortSegment(undefined)).toBe("default");
        expect(sortSegment({ field: "path", desc: true })).toBe("path:desc");
    });
});

describe("cycleSort", () => {
    it("cycles ascending, descending, then back to the backend default", () => {
        const first = cycleSort(null, "path");
        expect(first).toStrictEqual({ field: "path", desc: false });
        const second = cycleSort(first, "path");
        expect(second).toStrictEqual({ field: "path", desc: true });
        expect(cycleSort(second, "path")).toBeNull();
    });

    it("restarts ascending when the field changes", () => {
        const restarted = { field: "createdAt", desc: false };
        expect(cycleSort({ field: "path", desc: true }, "createdAt")).toStrictEqual(restarted);
        expect(cycleSort({ field: "path", desc: false }, "createdAt")).toStrictEqual(restarted);
    });
});
