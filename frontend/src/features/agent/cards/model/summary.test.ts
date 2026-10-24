import { describe, expect, it } from "vitest";

import { ownSummary } from "./summary.js";

const description = "Change how one field of a page type is written: on or off, instruction, lengths, link policy.";

describe("the confirmation's own words", () => {
    it("keeps what a call said of itself", () => {
        expect(ownSummary("The next run will rewrite these pages' own text: /a/, /b/.", description)).toBe(
            "The next run will rewrite these pages' own text: /a/, /b/.",
        );
    });

    it("leaves out the tool's description, with or without its arguments", () => {
        const plain = description.slice(0, -1);
        expect(ownSummary(plain, description)).toBeNull();
        expect(ownSummary(`${plain} with {"pageTypeId":"pt-hub"}`, description)).toBeNull();
    });

    it("never shows JSON, though the tool is not known yet", () => {
        expect(ownSummary('Delete a page with {"id":"p1"}', null)).toBeNull();
    });

    it("shows nothing for an empty summary", () => {
        expect(ownSummary("", description)).toBeNull();
        expect(ownSummary("   ", null)).toBeNull();
    });
});
