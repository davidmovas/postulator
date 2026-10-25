import { describe, expect, it } from "vitest";

import { runLayout } from "./columns.js";

const besideTheDockAt960 = 960 - 68 - 392 - 212;

describe("the columns the runs table shows at a width", () => {
    it.each([
        { width: 0, columns: ["kind", "status", "progress", "spend", "started", "lasted"] },
        { width: 1200, columns: ["kind", "status", "progress", "spend", "started", "lasted"] },
        { width: 738, columns: ["kind", "status", "progress", "spend", "started", "lasted"] },
        { width: 737, columns: ["kind", "status", "progress", "spend", "started"] },
        { width: 654, columns: ["kind", "status", "progress", "spend", "started"] },
        { width: 653, columns: ["kind", "status", "progress", "started"] },
        { width: 562, columns: ["kind", "status", "progress", "started"] },
        { width: 561, columns: ["kind", "status", "progress"] },
        { width: 454, columns: ["kind", "status", "progress"] },
        { width: besideTheDockAt960, columns: ["kind", "status", "progress"] },
    ])("drops the duration, the spend and the start, in that order, by $width px", ({ width, columns }) => {
        expect(runLayout(width).columns).toStrictEqual(columns);
    });

    it("lets the status and the progress shrink once even the three of them do not fit", () => {
        expect(runLayout(453).template).toBe("64px minmax(0,1.2fr) minmax(0,1fr)");
        expect(runLayout(besideTheDockAt960).least).toBeLessThanOrEqual(besideTheDockAt960);
    });

    it("keeps the full template while every column fits", () => {
        expect(runLayout(738).template).toBe("64px 192px minmax(150px,1.3fr) 80px 96px 72px");
    });
});
