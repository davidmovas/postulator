import { describe, expect, it } from "vitest";

import { bodyStates } from "../../generated/vocab.js";
import { bodyStateLabel, bodyStateTone, bodyStateTooltip } from "./labels.js";

describe("the words of a body's state", () => {
    it.each(bodyStates.map((state) => [state]))("%s has a word and one sentence", (state) => {
        expect(bodyStateLabel(state)).not.toBe(state);
        expect(bodyStateTooltip(state)).toMatch(/^[A-Z][^.]*\.$/);
    });

    it("keeps a state this build does not know as it came, untoned", () => {
        expect(bodyStateLabel("haunted")).toBe("haunted");
        expect(bodyStateTooltip("haunted")).toBe("");
        expect(bodyStateTone("haunted")).toBe("muted");
    });
});
