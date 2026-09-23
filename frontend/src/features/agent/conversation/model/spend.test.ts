import { describe, expect, test } from "vitest";

import { cachedPercent } from "./spend.js";

describe("the cached share of what a conversation read", () => {
    const cases: readonly { name: string; input: number; cachedInput: number; want: number }[] = [
        { name: "a real turn against a warm cache", input: 136307, cachedInput: 117224, want: 86 },
        { name: "nothing read yet", input: 0, cachedInput: 0, want: 0 },
        { name: "a first turn reads nothing from the cache", input: 20000, cachedInput: 0, want: 0 },
        { name: "everything came from the cache", input: 5000, cachedInput: 5000, want: 100 },
        { name: "a provider reporting more cached than read is held at the whole", input: 100, cachedInput: 140, want: 100 },
    ];

    for (const tc of cases) {
        test(tc.name, () => {
            expect(cachedPercent({ usd: 1, calls: 3, input: tc.input, cachedInput: tc.cachedInput })).toBe(tc.want);
        });
    }
});
