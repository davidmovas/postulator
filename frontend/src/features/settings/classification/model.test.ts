import { describe, expect, it } from "vitest";

import { copy } from "../../../copy/index.js";
import type { ClassificationGroup, RuleConditions } from "../../../data/types.js";
import { ruleConditions, ruleVerdicts, schemaTypes, urlModes } from "../../../generated/vocab.js";
import { conditionLabel, schemaOption, schemaWord, urlModeHint, urlModeLabel, verdictLabel } from "./labels.js";
import type { ConditionsDraft } from "./model.js";
import {
    asksAnything,
    blankConditions,
    closedOver,
    conditionsOf,
    draftOf,
    groupKey,
    keyOf,
    numberOf,
    pagesIn,
    placeOf,
    reordered,
    sentence,
} from "./model.js";

const said = copy.classification.sentence;

const names = {
    type: (id: string) => (id === "t-cat" ? "Category" : "Guide"),
    page: (id: string) => (id === "p-lab" ? "/lab-guides/" : "a page"),
};

function asConditions(held: Record<string, unknown>): RuleConditions {
    return held as RuleConditions;
}

describe("a rule in words", () => {
    it.each<{ label: string; conditions: Record<string, unknown>; words: string }>([
        { label: "no condition", conditions: {}, words: said.every },
        { label: "the front page", conditions: { frontPage: true }, words: "Pages that are the front page." },
        {
            label: "post types and an address",
            conditions: { postTypes: ["page", "post"], urls: [{ mode: "segment", value: "guides" }] },
            words: "Pages that are pages or posts and have an address that has the word guides.",
        },
        {
            label: "every address mode",
            conditions: {
                urls: [
                    { mode: "prefix", value: "/blog/" },
                    { mode: "segment", value: "guides" },
                    { mode: "glob", value: "/shop/**" },
                ],
            },
            words: "Pages that have an address that starts with /blog/, has the word guides or matches /shop/**.",
        },
        { label: "slugs", conditions: { slugs: ["about", "contact"] }, words: "Pages that have the slug about or contact." },
        { label: "under a page", conditions: { ancestor: { pageId: "p-lab" } }, words: "Pages that sit under /lab-guides/." },
        { label: "under a type", conditions: { ancestor: { typeId: "t-cat" } }, words: "Pages that sit under a Category page." },
        { label: "a level from", conditions: { level: { min: 2 } }, words: "Pages that are at level 2 or deeper." },
        { label: "a level to", conditions: { level: { max: 1 } }, words: "Pages that are at level 1 or nearer the top." },
        { label: "one level", conditions: { level: { min: 1, max: 1 } }, words: "Pages that are at level 1." },
        { label: "a level range", conditions: { level: { min: 1, max: 2 } }, words: "Pages that are at levels 1 to 2." },
        { label: "a listing", conditions: { listing: { present: true } }, words: "Pages that list products." },
        { label: "a listing of five", conditions: { listing: { present: true, minProducts: 5 } }, words: "Pages that list at least 5 products." },
        { label: "no listing", conditions: { listing: { present: false } }, words: "Pages that list no products." },
        { label: "schema", conditions: { schema: ["Article", "HowTo"] }, words: "Pages that are marked as an article or a how-to." },
        { label: "words", conditions: { words: { min: 300, max: 800 } }, words: "Pages that have 300 to 800 words." },
        { label: "H2 headings", conditions: { minH2: 3 }, words: "Pages that have at least 3 H2 headings." },
        {
            label: "every flag both ways",
            conditions: { technical: false, frontPage: false, shop: true },
            words: "Pages that are not technical pages, are not the front page and are the shop.",
        },
    ])("says $label", ({ conditions, words }) => {
        expect(sentence(asConditions(conditions), names)).toBe(words);
    });
});

describe("the form a rule is edited in", () => {
    const full: ConditionsDraft = {
        postTypes: ["page", " recipe ", "page", ""],
        urls: [
            { mode: "segment", value: " guides " },
            { mode: "prefix", value: "  " },
        ],
        slugs: ["about"],
        under: "type",
        underPage: "p-lab",
        underType: "t-cat",
        levelMin: "1",
        levelMax: "",
        listing: "present",
        minProducts: "5",
        schema: ["Article"],
        wordsMin: "",
        wordsMax: "800",
        minH2: "0",
        technical: "no",
        frontPage: "any",
        shop: "yes",
    };

    it("sends only what was filled, trimmed and once each", () => {
        expect(conditionsOf(full)).toStrictEqual({
            postTypes: ["page", "recipe"],
            urls: [{ mode: "segment", value: "guides" }],
            slugs: ["about"],
            ancestor: { typeId: "t-cat" },
            level: { min: 1 },
            listing: { present: true, minProducts: 5 },
            schema: ["Article"],
            words: { max: 800 },
            technical: false,
            shop: true,
        });
    });

    it("sends nothing for a blank form", () => {
        expect(conditionsOf(blankConditions)).toStrictEqual({});
        expect(asksAnything(conditionsOf(blankConditions))).toBe(false);
        expect(asksAnything(conditionsOf(full))).toBe(true);
    });

    it("reads a saved rule back into the form it was sent from", () => {
        const sent = conditionsOf(full);
        expect(conditionsOf(draftOf(sent))).toStrictEqual(sent);
        expect(draftOf(sent).under).toBe("type");
        expect(draftOf({ ancestor: { pageId: "p-lab" } }).under).toBe("page");
        expect(draftOf(null)).toBe(blankConditions);
    });

    it.each([
        { raw: "", value: undefined },
        { raw: " 12 ", value: 12 },
        { raw: "2.7", value: 2 },
        { raw: "many", value: undefined },
    ])("reads $raw as a number", ({ raw, value }) => {
        expect(numberOf(raw)).toBe(value);
    });

    it.each([
        { field: "conditions.urls[1].value", place: { condition: "urls", index: 1 } },
        { field: "conditions.level", place: { condition: "level", index: null } },
        { field: "typeId", place: null },
        { field: null, place: null },
    ])("places the refusal of $field", ({ field, place }) => {
        expect(placeOf(field)).toStrictEqual(place);
    });
});

describe("moving a rule", () => {
    it.each<{ id: string; delta: -1 | 1; order: string[] | null }>([
        { id: "b", delta: -1, order: ["b", "a", "c"] },
        { id: "b", delta: 1, order: ["a", "c", "b"] },
        { id: "a", delta: -1, order: null },
        { id: "c", delta: 1, order: null },
        { id: "z", delta: 1, order: null },
    ])("moves $id by $delta", ({ id, delta, order }) => {
        expect(reordered(["a", "b", "c"], id, delta)).toStrictEqual(order);
    });
});

describe("choosing the changes to apply", () => {
    const group = (from: string, to: string, pages: number, dependsOn: { from: string; to: string }[] = []): ClassificationGroup => ({
        from,
        to,
        pages,
        retyped: true,
        rules: [],
        dependsOn,
        samples: [],
    });
    const groups = [group("", "t-hub", 3), group("t-entity", "t-cat", 20, [{ from: "", to: "t-hub" }]), group("t-entity", "t-guide", 90, [{ from: "t-entity", to: "t-cat" }])];
    const key = (from: string, to: string): string => groupKey({ from, to });

    it("ticks what a change depends on, all the way up", () => {
        expect([...closedOver(new Set(), groups, key("t-entity", "t-guide"), true)].sort()).toStrictEqual(
            [key("t-entity", "t-guide"), key("t-entity", "t-cat"), key("", "t-hub")].sort(),
        );
    });

    it("unticks what depends on a change taken away", () => {
        const all = new Set(groups.map(groupKey));
        expect([...closedOver(all, groups, key("", "t-hub"), false)]).toStrictEqual([]);
        expect([...closedOver(all, groups, key("t-entity", "t-guide"), false)].sort()).toStrictEqual([key("", "t-hub"), key("t-entity", "t-cat")].sort());
    });

    it("counts the pages of the chosen changes, or of all", () => {
        expect(pagesIn(groups, null)).toBe(113);
        expect(pagesIn(groups, new Set([key("", "t-hub")]))).toBe(3);
    });

    it("reads a key back into the group it names", () => {
        expect(keyOf(key("", "t-hub"))).toStrictEqual({ from: "", to: "t-hub" });
    });
});

describe("the classification vocabulary in words", () => {
    it.each<{ name: string; values: readonly string[]; label: (value: string) => string }>([
        { name: "ruleConditions", values: ruleConditions, label: conditionLabel },
        { name: "ruleVerdicts", values: ruleVerdicts, label: verdictLabel },
        { name: "urlModes", values: urlModes, label: urlModeLabel },
        { name: "urlModes hints", values: urlModes, label: urlModeHint },
        { name: "schemaTypes", values: schemaTypes, label: schemaWord },
        { name: "schemaTypes options", values: schemaTypes, label: schemaOption },
    ])("names every one of $name", ({ values, label }) => {
        for (const value of values) {
            expect(label(value), value).not.toBe("");
        }
    });
});
