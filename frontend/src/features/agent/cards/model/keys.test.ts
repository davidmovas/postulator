import { describe, expect, it } from "vitest";

import { cardChoice } from "./keys.js";

describe("cardChoice", () => {
    const stroke = (key: string, ctrlKey = false, metaKey = false, shiftKey = false) => ({
        key,
        ctrlKey,
        metaKey,
        shiftKey,
    });

    it("approves on Ctrl+Enter and on Cmd+Enter in the transcript", () => {
        expect(cardChoice("confirm", stroke("Enter", true))).toBe("approve");
        expect(cardChoice("confirm", stroke("Enter", false, true))).toBe("approve");
    });

    it("rejects on the same stroke held with Shift, so a refusal is as quick as an approval", () => {
        expect(cardChoice("confirm", stroke("Enter", true, false, true))).toBe("reject");
        expect(cardChoice("confirm", stroke("Enter", false, true, true))).toBe("reject");
    });

    it("never decides on Escape, which closes the top-most overlay", () => {
        expect(cardChoice("confirm", stroke("Escape"))).toBeNull();
        expect(cardChoice("list", stroke("Escape"))).toBeNull();
        expect(cardChoice("none", stroke("Escape"))).toBeNull();
    });

    it("leaves a plain Enter to the composer and ignores every other key", () => {
        expect(cardChoice("confirm", stroke("Enter"))).toBeNull();
        expect(cardChoice("confirm", stroke("a"))).toBeNull();
        expect(cardChoice("confirm", stroke("r"))).toBeNull();
    });

    it("lets the list own its own keys", () => {
        expect(cardChoice("list", stroke("Enter", true))).toBeNull();
        expect(cardChoice("none", stroke("Enter", true))).toBeNull();
    });
});
