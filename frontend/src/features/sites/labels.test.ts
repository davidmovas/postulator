import { describe, expect, it } from "vitest";

import { siteStatuses } from "../../generated/vocab.js";
import { siteStatusLabel } from "./status.js";

describe("the site vocabulary in words", () => {
    it("names every site status without printing the stored value", () => {
        for (const status of siteStatuses) {
            const label = siteStatusLabel(status);
            expect(label).not.toBe("");
            expect(label).not.toBe(status);
        }
    });

    it("shows a status it has never heard of as it arrived", () => {
        expect(siteStatusLabel("melted")).toBe("melted");
    });
});
