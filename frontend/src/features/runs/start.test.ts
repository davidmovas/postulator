import { describe, expect, it } from "vitest";

import { runKinds, runKindsWithTheirOwnRecipe } from "../../generated/vocab.js";
import { kindDoes, takesATemplate, tokenCapOf } from "./start.js";

describe("what the drawer says a kind does", () => {
    it("has one sentence for every kind a run can be", () => {
        for (const kind of runKinds) {
            const said = kindDoes(kind);
            expect(said, kind).not.toBe("");
            expect(said, kind).toMatch(/\.$/);
        }
    });

    it("says nothing about a kind this build does not know", () => {
        expect(kindDoes("teleport")).toBe("");
    });

    it("tells a relink apart from a generate", () => {
        expect(kindDoes("relink")).not.toBe(kindDoes("generate"));
        expect(kindDoes("relink")).toMatch(/no model/);
        expect(kindDoes("repair")).toMatch(/No model/);
    });
});

describe("the token cap the drawer sends", () => {
    const cases: readonly (readonly [string, number])[] = [
        ["0", 0],
        ["", 0],
        ["   ", 0],
        ["-40", 0],
        ["abc", 0],
        ["120000", 120000],
        ["12_000", 12],
        ["4.9", 4],
    ];

    for (const [typed, want] of cases) {
        it(`reads ${JSON.stringify(typed)} as ${want}`, () => {
            expect(tokenCapOf(typed)).toBe(want);
        });
    }
});

describe("the template select the drawer offers", () => {
    it("is offered for a kind that takes a template's recipe", () => {
        for (const kind of runKinds) {
            if ((runKindsWithTheirOwnRecipe as readonly string[]).includes(kind)) {
                continue;
            }
            expect(takesATemplate(kind), kind).toBe(true);
        }
    });

    it("is hidden for every kind that names its own steps", () => {
        expect(runKindsWithTheirOwnRecipe.length).toBeGreaterThan(0);
        for (const kind of runKindsWithTheirOwnRecipe) {
            expect(takesATemplate(kind), kind).toBe(false);
        }
    });
});

