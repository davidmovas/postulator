import { describe, expect, it } from "vitest";

import { copyName } from "./naming.js";

describe("copyName", () => {
    const cases: readonly { name: string; source: string; taken: readonly string[]; want: string }[] = [
        { name: "the first copy takes the plain name", source: "Guide", taken: ["Guide"], want: "Guide copy" },
        {
            name: "a second copy is numbered",
            source: "Guide",
            taken: ["Guide", "Guide copy"],
            want: "Guide copy 2",
        },
        {
            name: "numbering keeps going past a gap",
            source: "Guide",
            taken: ["Guide", "Guide copy", "Guide copy 2", "Guide copy 3"],
            want: "Guide copy 4",
        },
        {
            name: "the unique index ignores case, so the name does too",
            source: "Guide",
            taken: ["guide COPY"],
            want: "Guide copy 2",
        },
        {
            name: "surrounding space in a stored name still counts as taken",
            source: "Guide",
            taken: ["  Guide copy  "],
            want: "Guide copy 2",
        },
        { name: "an empty workspace needs no number", source: "Hub", taken: [], want: "Hub copy" },
        {
            name: "a copy of a copy reads as one",
            source: "Guide copy",
            taken: ["Guide copy"],
            want: "Guide copy copy",
        },
    ];

    for (const held of cases) {
        it(held.name, () => {
            expect(copyName(held.source, held.taken)).toBe(held.want);
        });
    }

    it("never answers with a name that is already taken", () => {
        const taken = ["Guide", "Guide copy", ...Array.from({ length: 50 }, (_unused, at) => `Guide copy ${at + 2}`)];
        expect(taken).not.toContain(copyName("Guide", taken));
    });
});
