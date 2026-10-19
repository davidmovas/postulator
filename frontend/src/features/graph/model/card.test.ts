import { describe, expect, it } from "vitest";

import { cardChanged, cardOf, cardProblems, cardRequest, hasProblems } from "./card.js";
import { entity } from "./fixture.js";

const carded = {
    ...entity({ id: "bpc", name: "BPC-157" }),
    definition: "A peptide studied for tissue repair.",
    facts: [{ label: "Form", value: "Lyophilised powder" }],
    notes: [{ label: "Tone", text: "Research use only." }],
};

describe("the entity card draft", () => {
    it("starts from what the entity holds and has nothing to send", () => {
        const draft = cardOf(carded);
        expect(draft.facts).toStrictEqual([{ label: "Form", value: "Lyophilised powder" }]);
        expect(cardChanged(carded, draft)).toBe(false);
        expect(cardRequest(carded, draft)).toBeNull();
    });

    it("reads an entity with no card as an empty one", () => {
        const bare = { ...entity({ id: "e1", name: "Bare" }), facts: null, notes: null };
        expect(cardOf(bare)).toStrictEqual({ definition: "", facts: [], notes: [] });
    });

    it("refuses a fact or a note with one half blank, as Go would", () => {
        const draft = {
            ...cardOf(carded),
            facts: [...cardOf(carded).facts, { label: "Purity", value: "  " }, { label: "", value: "" }],
            notes: [{ label: "", text: "Only the text" }],
        };
        const problems = cardProblems(draft);
        expect([...problems.facts]).toStrictEqual([1]);
        expect([...problems.notes]).toStrictEqual([0]);
        expect(hasProblems(problems)).toBe(true);
        expect(hasProblems(cardProblems(cardOf(carded)))).toBe(false);
    });

    it("sends only the parts that changed, trimmed, with rows left wholly empty dropped", () => {
        const draft = {
            ...cardOf(carded),
            facts: [...cardOf(carded).facts, { label: " Purity ", value: " 99% " }, { label: " ", value: "" }],
        };
        expect(cardRequest(carded, draft)).toStrictEqual({
            id: "bpc",
            definition: null,
            facts: [
                { label: "Form", value: "Lyophilised powder" },
                { label: "Purity", value: "99%" },
            ],
            notes: null,
        });
    });

    it("clears a list by sending it empty, and a definition by sending it blank", () => {
        const draft = { ...cardOf(carded), definition: "  ", notes: [] };
        expect(cardRequest(carded, draft)).toStrictEqual({ id: "bpc", definition: "", facts: null, notes: [] });
    });

    it("sends nothing when the only change is an empty row", () => {
        const draft = { ...cardOf(carded), notes: [...cardOf(carded).notes, { label: "", text: "" }] };
        expect(cardChanged(carded, draft)).toBe(true);
        expect(cardRequest(carded, draft)).toBeNull();
    });
});
