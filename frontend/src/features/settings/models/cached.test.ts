import { describe, expect, test } from "vitest";

import type { UsageSummary } from "./cached.js";
import { cachedShare } from "./cached.js";

function summary(input: number, cachedInput: number): UsageSummary {
    return { usd: 0.42, calls: 7, usage: { input, cachedInput, total: input + 900 } };
}

describe("the cached share the spend tile shows", () => {
    const cases: readonly { name: string; input: number; cachedInput: number; want: number | null }[] = [
        { name: "nothing has been spent yet", input: 0, cachedInput: 0, want: null },
        { name: "a first call reads nothing from the cache", input: 20000, cachedInput: 0, want: null },
        { name: "a warm cache", input: 136307, cachedInput: 117224, want: 86 },
        { name: "everything came from the cache", input: 5000, cachedInput: 5000, want: 100 },
        { name: "a provider reporting more cached than read", input: 100, cachedInput: 140, want: 100 },
    ];

    for (const tc of cases) {
        test(tc.name, () => {
            expect(cachedShare(summary(tc.input, tc.cachedInput))).toBe(tc.want);
        });
    }
});
