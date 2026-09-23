import { describe, expect, it } from "vitest";

import type { JsonObject } from "../../domain/merge-patch.js";
import { stepNames } from "../../generated/vocab.js";
import { draftFromJson, moved, specJsonOf } from "./spec.js";

const hub: JsonObject = {
    sections: [
        {
            heading: "Overview",
            intent: "Define the topic",
            targetWords: 250,
            required: true,
            keywordRules: { include: [], primaryInHeading: true },
        },
        {
            heading: "Key Areas",
            intent: "Introduce every child topic",
            targetWords: 700,
            required: true,
            keywordRules: { include: ["mugs"], primaryInHeading: false },
        },
    ],
    tone: "Authoritative and plain",
    length: { min: 1600, max: 2400 },
    keywordRules: { primaryInTitle: true, primaryInH1: true, primaryInFirstParagraph: true, maxDensity: 0.025 },
    linkRules: {
        upDepth: 1,
        downLinks: true,
        siblingMinWeight: 0.5,
        maxLinks: 20,
        maxPerTarget: 1,
        parentLinkWithinParagraphs: 2,
        childrenSection: true,
    },
    metaRules: { titlePattern: "{primaryKeyword} | {siteName}", descriptionMax: 155 },
    images: { featured: true, inline: 0, source: "ai" },
    modelProfiles: {},
    recipe: [
        { name: "resolve_context", enabled: true },
        { name: "generate_body", enabled: true },
        { name: "publish", enabled: true, params: { refuseDrift: true } },
        { name: "report", enabled: true },
    ],
};

describe("template spec drafts", () => {
    it("reads every part of a stored spec", () => {
        const draft = draftFromJson(hub);
        expect(draft.sections).toHaveLength(2);
        expect(draft.sections[1]?.include).toStrictEqual(["mugs"]);
        expect(draft.lengthMin).toBe(1600);
        expect(draft.maxDensity).toBe(0.025);
        expect(draft.upDepth).toBe(1);
        expect(draft.imageSource).toBe("ai");
        expect(draft.profiles).toStrictEqual([]);
    });

    it("lists the recipe in pipeline order with the declared steps switched on", () => {
        const draft = draftFromJson(hub);
        expect(draft.recipe.map((step) => step.name)).toStrictEqual([...stepNames]);
        expect(draft.recipe.filter((step) => step.enabled).map((step) => step.name)).toStrictEqual([
            "resolve_context",
            "generate_body",
            "publish",
            "report",
        ]);
        expect(draft.recipe.find((step) => step.name === "judge")?.declared).toBe(false);
    });

    it("keeps a step the pipeline does not know about", () => {
        const draft = draftFromJson({ ...hub, recipe: [{ name: "invent_slogans", enabled: true }] });
        const step = draft.recipe.at(-1);
        expect(step?.name).toBe("invent_slogans");
        expect(step?.enabled).toBe(true);
    });

    it("rebuilds the same document it read", () => {
        expect(specJsonOf(draftFromJson(hub))).toStrictEqual(hub);
    });

    it("keeps the params a step carries when the step is switched off", () => {
        const draft = draftFromJson(hub);
        const recipe = draft.recipe.map((step) => (step.name === "publish" ? { ...step, enabled: false } : step));
        const rebuilt = specJsonOf({ ...draft, recipe });
        expect(rebuilt["recipe"]).toStrictEqual([
            { name: "resolve_context", enabled: true },
            { name: "generate_body", enabled: true },
            { name: "publish", enabled: false, params: { refuseDrift: true } },
            { name: "report", enabled: true },
        ]);
    });

    it("writes a newly enabled step into pipeline order", () => {
        const draft = draftFromJson(hub);
        const recipe = draft.recipe.map((step) => (step.name === "judge" ? { ...step, enabled: true } : step));
        const rebuilt = specJsonOf({ ...draft, recipe });
        expect(rebuilt["recipe"]).toStrictEqual([
            { name: "resolve_context", enabled: true },
            { name: "generate_body", enabled: true },
            { name: "judge", enabled: true },
            { name: "publish", enabled: true, params: { refuseDrift: true } },
            { name: "report", enabled: true },
        ]);
    });

    it("survives a document a removal emptied out", () => {
        const draft = draftFromJson({ tone: "plain" });
        expect(draft.sections).toStrictEqual([]);
        expect(draft.lengthMin).toBe(0);
        expect(draft.recipe.every((step) => !step.enabled)).toBe(true);
        expect(draft.imageSource).toBe("");
    });

    it("orders pinned roles the way the vocabulary does and keeps an unknown one", () => {
        const draft = draftFromJson({
            ...hub,
            modelProfiles: {
                narrator: { provider: "openai", model: "gpt" },
                editor: { provider: "anthropic", model: "sonnet" },
                writer: { provider: "anthropic", model: "opus" },
            },
        });
        expect(draft.profiles.map((profile) => profile.role)).toStrictEqual(["writer", "editor", "narrator"]);
    });
});

describe("moved", () => {
    it("moves an item forward", () => {
        expect(moved(["a", "b", "c"], 0, 2)).toStrictEqual(["b", "c", "a"]);
    });

    it("moves an item backward", () => {
        expect(moved(["a", "b", "c"], 2, 0)).toStrictEqual(["c", "a", "b"]);
    });

    it("refuses a position off the end", () => {
        expect(moved(["a", "b"], 0, 5)).toStrictEqual(["a", "b"]);
    });
});
