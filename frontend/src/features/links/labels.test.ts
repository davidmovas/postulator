import { describe, expect, it } from "vitest";

import { linkBlockedReasons } from "../../generated/vocab.js";
import { blockedReasonLabel } from "./labels.js";

describe("blocked reasons", () => {
    it.each(linkBlockedReasons)("words %s for the screen", (reason) => {
        const label = blockedReasonLabel(reason);
        expect(label).not.toBe("");
        expect(label).not.toBe(reason);
    });

    it("leaves an unknown reason as it came", () => {
        expect(blockedReasonLabel("invented")).toBe("invented");
    });
});
