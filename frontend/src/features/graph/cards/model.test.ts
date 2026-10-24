import { describe, expect, it } from "vitest";

import type { CardProposal } from "../../../data/types.js";
import { currentFacts, definitionMoves, proposedFacts, refusalOf, waitingEntities } from "./model.js";

function proposal(overrides: Partial<CardProposal> = {}): CardProposal {
    return {
        findingId: "f1",
        proposalHash: "seen-f1",
        entityId: "e1",
        name: "Peptides",
        current: {
            definition: "Short chains of amino acids.",
            facts: [
                { label: "Storage", value: "Keep cold" },
                { label: "Form", value: "Powder" },
            ],
        },
        proposed: {
            definition: "Short chains of amino acids.",
            facts: [
                { label: "storage", value: "Keep it at 2 to 8 °C" },
                { label: "Purity", value: "Above 98%" },
            ],
        },
        reason: "",
        changed: false,
        createdAt: "2026-10-19T09:00:00Z",
        ...overrides,
    };
}

describe("a card proposal set against the card", () => {
    it("marks each proposed fact as new or changed by its label, as the store dedupes it", () => {
        expect(proposedFacts(proposal()).map((fact) => [fact.label, fact.mark])).toStrictEqual([
            ["storage", "changed"],
            ["Purity", "added"],
        ]);
    });

    it("marks each current fact the proposal changes or drops", () => {
        expect(currentFacts(proposal()).map((fact) => [fact.label, fact.mark])).toStrictEqual([
            ["Storage", "changed"],
            ["Form", "removed"],
        ]);
    });

    it("leaves a fact both cards carry unmarked", () => {
        const same = proposal({ proposed: { definition: "Other.", facts: [{ label: "Form", value: " Powder " }] } });
        expect(proposedFacts(same).map((fact) => fact.mark)).toStrictEqual(["same"]);
    });

    it("says whether the definition moves", () => {
        expect(definitionMoves(proposal())).toBe(false);
        expect(definitionMoves(proposal({ proposed: { definition: "Peptides, sold for research.", facts: null } }))).toBe(true);
    });

    it("reads a card with no facts", () => {
        const bare = proposal({ current: { definition: "", facts: null }, proposed: { definition: "New.", facts: null } });
        expect(currentFacts(bare)).toStrictEqual([]);
        expect(proposedFacts(bare)).toStrictEqual([]);
    });

    it("names the entities a proposal waits for", () => {
        expect([...waitingEntities([proposal(), proposal({ findingId: "f2", entityId: "e2" })])]).toStrictEqual(["e1", "e2"]);
    });
});

describe("why an accept was refused", () => {
    it.each([
        ["CONFLICT", "card_changed", "changed"],
        ["CONFLICT", "proposal_changed", "replaced"],
        ["CONFLICT", undefined, "settled"],
        ["NOT_FOUND", undefined, "other"],
    ] as const)("reads %s with reason %s as %s", (code, reason, want) => {
        expect(refusalOf(code, reason)).toBe(want);
    });
});
