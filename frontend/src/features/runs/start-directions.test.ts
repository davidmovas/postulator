import { describe, expect, it } from "vitest";

import type { TransportError } from "../../data/errors.js";
import type { DirectiveDraft } from "./start-directions.js";
import {
    blockedByDrafts,
    briefOf,
    briefProblems,
    directiveOf,
    directiveProblems,
    directivesOf,
    directiveSize,
    emptyBrief,
    emptyDirective,
    filledOf,
    limits,
    mustLinkRefusal,
} from "./start-directions.js";
import { startRefusal, startRequestOf } from "./start-model.js";

function refused(cause: TransportError): Error {
    return Object.assign(new Error(cause.message), { cause });
}

function draft(overrides: Partial<DirectiveDraft>): DirectiveDraft {
    return { ...emptyDirective, ...overrides };
}

describe("the brief the drawer sends", () => {
    it("sends nothing for an empty brief", () => {
        expect(briefOf(emptyBrief)).toBeUndefined();
        expect(filledOf(emptyBrief)).toBe(0);
    });

    it("sends only the lines the person filled, trimmed", () => {
        expect(briefOf({ ...emptyBrief, goal: "  Sell the kits ", styleSample: "Short and plain." })).toStrictEqual({
            goal: "Sell the kits",
            styleSample: "Short and plain.",
        });
    });

    it("marks a line over its limit, counted in characters rather than bytes", () => {
        const long = "é".repeat(limits.line);
        expect(briefProblems({ ...emptyBrief, goal: long }).size).toBe(0);
        expect([...briefProblems({ ...emptyBrief, goal: `${long}é` })]).toStrictEqual(["goal"]);
        expect([...briefProblems({ ...emptyBrief, reference: "a".repeat(limits.reference + 1) })]).toStrictEqual([
            "reference",
        ]);
    });
});

describe("a page's directive", () => {
    it("is left out when nothing was said for the page", () => {
        expect(directiveOf("p1", undefined)).toBeNull();
        expect(directiveOf("p1", draft({ notes: "   ", keywords: [" "] }))).toBeNull();
    });

    it("carries what was said, with an empty anchor left to the linker", () => {
        expect(
            directiveOf(
                "p1",
                draft({
                    notes: "Lead with the starter kit",
                    addSections: ["Who it is for"],
                    keywords: ["peptide kit"],
                    mustLink: [
                        { pageId: "p2", path: "/peptides/", anchor: " peptides " },
                        { pageId: "p3", path: "/guide/", anchor: "" },
                    ],
                    minWords: "600",
                    maxWords: "",
                }),
            ),
        ).toStrictEqual({
            pageId: "p1",
            notes: "Lead with the starter kit",
            addSections: ["Who it is for"],
            keywords: ["peptide kit"],
            mustLink: [{ pageId: "p2", anchor: "peptides" }, { pageId: "p3" }],
            minWords: 600,
        });
    });

    it("sends the directives of the chosen pages only", () => {
        const drafts = new Map([
            ["p1", draft({ notes: "one" })],
            ["p9", draft({ notes: "not chosen" })],
        ]);
        expect(directivesOf(drafts, ["p1", "p2"])).toStrictEqual([{ pageId: "p1", notes: "one" }]);
    });

    it.each([
        ["notes over 2000 characters", draft({ notes: "a".repeat(limits.notes + 1) }), "notes"],
        ["eleven sections", draft({ addSections: Array.from({ length: 11 }, (_, at) => `S${at}`) }), "addSections"],
        ["a keyword over 200 characters", draft({ keywords: ["k".repeat(limits.item + 1)] }), "keywords"],
        ["words past the ceiling", draft({ minWords: "20001" }), "minWords"],
        ["words that are not a number", draft({ maxWords: "many" }), "maxWords"],
        ["the most below the least", draft({ minWords: "900", maxWords: "300" }), "wordsOrder"],
    ])("marks %s", (_name, held, problem) => {
        expect([...directiveProblems(held)]).toStrictEqual([problem]);
    });

    it("counts what a directive says and blocks the drawer on a problem", () => {
        const held = draft({ notes: "x", keywords: ["a", "b"], maxWords: "800" });
        expect(directiveSize(held)).toBe(4);
        expect(blockedByDrafts(emptyBrief, new Map([["p1", held]]), ["p1"])).toBe(false);
        expect(blockedByDrafts(emptyBrief, new Map([["p1", draft({ minWords: "-1" })]]), ["p1"])).toBe(true);
    });
});

describe("a must-link the graph does not connect", () => {
    const thrown = refused({
        code: "INVALID",
        message: "/kits/ does not link to /peptides/: no edge of the entity graph connects their entities",
        details: { field: "directives.mustLink", from: "/kits/", to: "/peptides/" },
    });

    it("is read with the two pages it names", () => {
        expect(mustLinkRefusal(thrown)).toStrictEqual({
            message: "/kits/ does not link to /peptides/: no edge of the entity graph connects their entities",
            from: "/kits/",
            to: "/peptides/",
        });
    });

    it("is said by its own notice rather than the drawer's banner", () => {
        expect(startRefusal(thrown)).toStrictEqual({ targets: null, banner: null });
    });

    it("is no must-link refusal when another field was refused", () => {
        const other = refused({ code: "INVALID", message: "too long", details: { field: "brief.goal" } });
        expect(mustLinkRefusal(other)).toBeNull();
    });
});

describe("the start request with a brief and directives", () => {
    const form = {
        siteId: "s1",
        pageIds: ["p1"],
        publishMode: "draft",
        kind: "generate",
        templateId: "",
        cap: "5",
        tokenCap: "0",
        rewrite: [],
        fields: [],
    };

    it("carries the brief, the directives and the edge offer", () => {
        const request = startRequestOf({
            ...form,
            brief: { goal: "Sell" },
            directives: [{ pageId: "p1", mustLink: [{ pageId: "p2" }] }],
            addEdges: true,
        });
        expect(request.brief).toStrictEqual({ goal: "Sell" });
        expect(request.directives).toStrictEqual([{ pageId: "p1", mustLink: [{ pageId: "p2" }] }]);
        expect(request.addEdges).toBe(true);
    });

    it("sends no edge offer without a directive", () => {
        expect(startRequestOf({ ...form, directives: [], addEdges: true })).not.toHaveProperty("addEdges");
    });
});
