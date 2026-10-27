import { describe, expect, it } from "vitest";

import type { ParentWhy } from "../../data/types.js";
import { bodyStates, parentWhyReasons } from "../../generated/vocab.js";
import { bodyStateLabel, bodyStateTone, bodyStateTooltip, whyLabel, whySentence } from "./labels.js";

function why(reason: string, facts: Partial<ParentWhy> = {}): ParentWhy {
    return { reason: reason as ParentWhy["reason"], ...facts };
}

describe("why a page has no parent", () => {
    it.each(parentWhyReasons.map((reason) => [reason]))("%s has a short label and one plain sentence", (reason) => {
        expect(whyLabel(why(reason))).not.toContain(reason);
        expect(whySentence(why(reason))).toMatch(/^[A-Z].*\.$/);
    });

    it.each([
        [
            why("crumb_unresolved", { crumb: "/gone/" }),
            "Its breadcrumb points at /gone/, which is no page in the model.",
        ],
        [
            why("crumb_ambiguous", { crumb: "Kits", candidates: ["/kits/", "/shop/kits/"] }),
            "Its breadcrumb names “Kits”, which 2 pages carry: /kits/, /shop/kits/. None was taken.",
        ],
        [
            why("cycle_broken", { loop: ["/a/", "/b/"] }),
            "Its parents looped through /a/ › /b/, so the loop was broken here.",
        ],
        [
            why("parent_archived", { parent: "/old/" }),
            "Its parent /old/ was taken off the site, so it sits at the top.",
        ],
    ])("names what it knows: %j", (held, sentence) => {
        expect(whySentence(held)).toBe(sentence);
    });

    it("labels the front page as itself and any other as having no parent", () => {
        expect(whyLabel(why("root"))).toBe("Front page");
        expect(whyLabel(why("no_signal"))).toBe("No parent · Nothing places it");
        expect(whyLabel(null)).toBe("No parent");
        expect(whySentence(why("haunted"))).toBeNull();
    });
});

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
