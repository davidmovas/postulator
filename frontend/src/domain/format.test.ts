import { describe, expect, it } from "vitest";

import { duration } from "./format.js";

describe("duration", () => {
    it("never writes sixty seconds, which is a minute", () => {
        expect(duration(299_700)).toBe("5m 0s");
        expect(duration(119_600)).toBe("2m 0s");
    });

    it("writes the units a run is measured in", () => {
        expect(duration(420)).toBe("420 ms");
        expect(duration(20_000)).toBe("20.0 s");
        expect(duration(95_000)).toBe("1m 35s");
    });
});
