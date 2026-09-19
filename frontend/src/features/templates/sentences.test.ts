import { describe, expect, it } from "vitest";

import type { JsonObject } from "../../domain/merge-patch.js";
import { sentencesOf } from "./sentences.js";
import { draftFromJson } from "./spec.js";

const base: JsonObject = {
    sections: [
        {
            heading: "Overview",
            intent: "Define the topic",
            targetWords: 250,
            required: true,
            keywordRules: { include: [], primaryInHeading: true },
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
    metaRules: { titlePattern: "{primaryKeyword}", descriptionMax: 155 },
    images: { featured: true, inline: 0, source: "ai" },
    modelProfiles: {},
    recipe: [
        { name: "resolve_context", enabled: true },
        { name: "generate_body", enabled: true },
        { name: "judge", enabled: true },
    ],
};

const draft = draftFromJson(base);

describe("sentencesOf", () => {
    it("says nothing without a patch", () => {
        expect(sentencesOf(draft, null)).toStrictEqual([]);
    });

    it("says the site shortened the page", () => {
        expect(sentencesOf(draft, { length: { min: 600, max: 900 } })).toStrictEqual([
            "shortens the page to 600–900 words",
        ]);
    });

    it("says the site lengthened the page", () => {
        expect(sentencesOf(draft, { length: { min: 1600, max: 3000 } })).toStrictEqual([
            "lengthens the page to 1600–3000 words",
        ]);
    });

    it("reads a half-open length change", () => {
        expect(sentencesOf(draft, { length: { max: 900 } })).toStrictEqual(["caps the page at 900 words"]);
    });

    it("renders keyword and link rules as prose", () => {
        expect(
            sentencesOf(draft, { keywordRules: { maxDensity: 0.015 }, linkRules: { maxLinks: 8, downLinks: false } }),
        ).toStrictEqual([
            "caps keyword density at 1.5%",
            "stops linking down to child pages",
            "allows at most 8 internal links",
        ]);
    });

    it("names the sections a replacement leaves behind", () => {
        expect(
            sentencesOf(draft, {
                sections: [
                    { heading: "Intro", intent: "", targetWords: 100, required: true, keywordRules: { include: [], primaryInHeading: false } },
                    { heading: "Buying guide", intent: "", targetWords: 400, required: true, keywordRules: { include: [], primaryInHeading: false } },
                ],
            }),
        ).toStrictEqual(["writes its own 2 sections: Intro and Buying guide"]);
    });

    it("names the steps a recipe change switches off", () => {
        expect(
            sentencesOf(draft, {
                recipe: [
                    { name: "resolve_context", enabled: true },
                    { name: "generate_body", enabled: true },
                    { name: "judge", enabled: false },
                ],
            }),
        ).toStrictEqual(["skips Score the page"]);
    });

    it("reads a pinned and an unpinned role", () => {
        expect(sentencesOf(draft, { modelProfiles: { writer: { provider: "anthropic", model: "opus" } } })).toStrictEqual(
            ["uses anthropic opus for the writer"],
        );
        const pinned = draftFromJson({ ...base, modelProfiles: { judge: { provider: "openai", model: "mini" } } });
        expect(sentencesOf(pinned, { modelProfiles: { judge: null } })).toStrictEqual([
            "lets the judge fall back to the site model",
        ]);
    });

    it("reads images and meta", () => {
        expect(sentencesOf(draft, { images: { inline: 3, source: "wpmedia" }, metaRules: { descriptionMax: 120 } })).toStrictEqual(
            [
                "asks for 3 inline images",
                "takes images from the WordPress media library",
                "caps the meta description at 120 characters",
            ],
        );
    });

    it("names a key it has no phrasing for", () => {
        expect(sentencesOf(draft, { somethingNew: 1 })).toStrictEqual(["changes somethingNew"]);
    });
});
