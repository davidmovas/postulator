import { describe, expect, it } from "vitest";

import { cadenceWords } from "./cron.js";

describe("cadenceWords over an interval", () => {
    it.each([
        [1, "Every minute"],
        [5, "Every 5 minutes"],
        [30, "Every 30 minutes"],
        [60, "Every hour"],
        [360, "Every 6 hours"],
        [1440, "Every day"],
        [2880, "Every 2 days"],
        [10080, "Every 7 days"],
        [90, "Every 90 minutes"],
    ])("reads %i minutes as %s", (minutes, want) => {
        expect(cadenceWords("", minutes)).toBe(want);
    });
});

describe("cadenceWords over a cron expression", () => {
    it.each([
        ["0 3 * * *", "Every day at 03:00 UTC"],
        ["30 4 * * *", "Every day at 04:30 UTC"],
        ["0 6 * * 1", "Every Monday at 06:00 UTC"],
        ["0 6 * * MON", "Every Monday at 06:00 UTC"],
        ["0 5 * * 0", "Every Sunday at 05:00 UTC"],
        ["0 5 * * 7", "Every Sunday at 05:00 UTC"],
        ["0 6 * * 1-5", "Every weekday at 06:00 UTC"],
        ["0 6 * * 0,6", "Every weekend day at 06:00 UTC"],
        ["0 6 * * 1,3,5", "Every Monday, Wednesday and Friday at 06:00 UTC"],
        ["0 6 * * 2,4", "Every Tuesday and Thursday at 06:00 UTC"],
        ["0 3,15 * * *", "Every day at 03:00 and 15:00 UTC"],
        ["0 1 3 * *", "Every month on the 3rd at 01:00 UTC"],
        ["0 1 1 * *", "Every month on the 1st at 01:00 UTC"],
        ["0 1 22 * *", "Every month on the 22nd at 01:00 UTC"],
        ["0 1 1 1 *", "Every year on 1 January at 01:00 UTC"],
        ["0 1 24 DEC *", "Every year on 24 December at 01:00 UTC"],
        ["0 2 */3 * *", "Every 3 days at 02:00 UTC"],
        ["*/15 * * * *", "Every 15 minutes"],
        ["* * * * *", "Every minute"],
        ["0 * * * *", "Every hour"],
        ["20 * * * *", "Every hour at :20"],
        ["0 */6 * * *", "Every 6 hours"],
        ["10 */6 * * *", "Every 6 hours at :10"],
        ["@daily", "Every day at 00:00 UTC"],
        ["@midnight", "Every day at 00:00 UTC"],
        ["@hourly", "Every hour"],
        ["@weekly", "Every Sunday at 00:00 UTC"],
        ["@monthly", "Every month on the 1st at 00:00 UTC"],
        ["@yearly", "Every year on 1 January at 00:00 UTC"],
        ["@annually", "Every year on 1 January at 00:00 UTC"],
        ["@every 30m", "Every 30 minutes"],
        ["@every 1h30m", "Every 90 minutes"],
        ["@every 6h", "Every 6 hours"],
        ["@every 45s", "Every 45 seconds"],
        ["  0 3 * * *  ", "Every day at 03:00 UTC"],
    ])("reads %s as %s", (expression, want) => {
        expect(cadenceWords(expression, 0)).toBe(want);
    });
});

describe("cadenceWords falls back to what it was given", () => {
    it.each([
        ["0 0 1 1 1", "0 0 1 1 1"],
        ["0 3 * *", "0 3 * *"],
        ["every other tuesday", "every other tuesday"],
        ["0 1,2,3,4,5 * * *", "0 1,2,3,4,5 * * *"],
        ["@every tuesday", "@every tuesday"],
        ["61 3 * * *", "61 3 * * *"],
    ])("leaves %s alone", (expression, want) => {
        expect(cadenceWords(expression, 0)).toBe(want);
    });

    it("has nothing to say without a cadence", () => {
        expect(cadenceWords("", 0)).toBe("");
        expect(cadenceWords("   ", 0)).toBe("");
    });

    it("prefers the interval when both are given", () => {
        expect(cadenceWords("0 3 * * *", 30)).toBe("Every 30 minutes");
    });
});
