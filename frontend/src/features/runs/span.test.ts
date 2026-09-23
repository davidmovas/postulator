import { describe, expect, it } from "vitest";

import { spanMs } from "./span.js";

const now = Date.parse("2026-09-20T12:00:30Z");

describe("spanMs", () => {
    it("measures a finished run between its own two instants", () => {
        expect(spanMs("2026-09-20T11:59:00Z", "2026-09-20T11:59:45Z", now)).toBe(45_000);
    });

    it("measures a live run against the tick", () => {
        expect(spanMs("2026-09-20T12:00:00Z", null, now)).toBe(30_000);
        expect(spanMs("2026-09-20T12:00:00Z", "", now)).toBe(30_000);
    });

    it("never goes backwards when the clock disagrees with the row", () => {
        expect(spanMs("2026-09-20T12:00:40Z", null, now)).toBe(0);
    });

    it("has nothing to measure before a run starts", () => {
        expect(spanMs(null, null, now)).toBeNull();
        expect(spanMs("", null, now)).toBeNull();
        expect(spanMs("not a time", null, now)).toBeNull();
        expect(spanMs("2026-09-20T12:00:00Z", "not a time", now)).toBeNull();
    });
});
