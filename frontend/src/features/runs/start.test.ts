import { describe, expect, it } from "vitest";

import { runKinds, runKindsWithTheirOwnRecipe } from "../../generated/vocab.js";
import type { TransportError } from "../../lib/errors.js";
import { kindDoes, startRefusal, takesATemplate, tokenCapOf } from "./start.js";

function refused(cause: TransportError): Error {
    return Object.assign(new Error(cause.message), { cause });
}

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

describe("where the drawer puts a refusal", () => {
    it("says nothing when nothing was refused", () => {
        expect(startRefusal(null)).toEqual({ targets: null, banner: null });
        expect(startRefusal(undefined)).toEqual({ targets: null, banner: null });
    });

    it("puts a refusal naming pageIds under the page list", () => {
        const refusal = startRefusal(
            refused({
                code: "INVALID",
                message: "nothing is mapped to /hub/child/",
                details: { field: "pageIds", paths: ["/hub/child/"] },
            }),
        );
        expect(refusal.targets).toBe("nothing is mapped to /hub/child/");
        expect(refusal.banner).toBeNull();
    });

    it("puts a refusal naming another field in the banner", () => {
        const refusal = startRefusal(
            refused({
                code: "INVALID",
                message: "the step sync_site belongs to a run of its own",
                details: { field: "recipe" },
            }),
        );
        expect(refusal.targets).toBeNull();
        expect(refusal.banner).toMatch(/sync_site/);
    });

    it("keeps a lock and a cancellation out of both", () => {
        expect(startRefusal(refused({ code: "LOCKED", message: "locked" }))).toEqual({ targets: null, banner: null });
        expect(startRefusal(refused({ code: "CANCELLED", message: "gone" }))).toEqual({ targets: null, banner: null });
    });
});

