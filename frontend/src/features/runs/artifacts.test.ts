import { describe, expect, it } from "vitest";

import {
    actualLinks,
    countByClass,
    decodeArtifact,
    draftView,
    finalView,
    imagesView,
    judgeView,
    linkContextView,
    metaView,
    owedTargets,
    publishView,
    relinkView,
    syncView,
    validationView,
    weigh,
} from "./artifacts.js";

describe("decodeArtifact", () => {
    it("answers null rather than throwing on anything that is not JSON", () => {
        expect(decodeArtifact("")).toBeNull();
        expect(decodeArtifact("<html>")).toBeNull();
        expect(decodeArtifact("{")).toBeNull();
    });

    it("decodes a JSON object", () => {
        expect(decodeArtifact('{"score":0.8}')).toStrictEqual({ score: 0.8 });
    });
});

describe("linkContextView", () => {
    it("reads the targets the graph asked for", () => {
        const view = linkContextView({
            pageId: "p1",
            pageUrl: "/a/",
            entityId: "e1",
            targets: [
                { entityId: "e2", pageId: "p2", url: "/b/", anchors: ["b"], relation: "up", required: true },
                { entityId: "e3", pageId: "p3", url: "/c/", anchors: [], relation: "sibling", required: false },
            ],
        });
        expect(view?.targets).toHaveLength(2);
        expect(view?.targets[0].required).toBe(true);
        expect(view?.targets[1].anchors).toStrictEqual([]);
    });

    it("answers null for anything that is not an object", () => {
        expect(linkContextView(null)).toBeNull();
        expect(linkContextView([])).toBeNull();
        expect(linkContextView("nope")).toBeNull();
    });

    it("survives a target that is not an object", () => {
        const view = linkContextView({ targets: ["nope", 7, { url: "/b/" }] });
        expect(view?.targets).toHaveLength(1);
        expect(view?.targets[0].url).toBe("/b/");
    });
});

const validation = {
    pageId: "p1",
    score: 0.75,
    compliance: {
        score: 0.75,
        items: [
            {
                severity: "error",
                code: "target_missing",
                message: "the page does not link to /c/",
                details: { targetPageId: "p3", relation: "down" },
            },
            {
                severity: "warn",
                code: "unknown_internal_link",
                message: "the graph does not sanction /x/",
                details: { href: "/x/" },
            },
            {
                severity: "error",
                code: "self_link",
                message: "the page links to itself",
                details: { href: "/a/" },
            },
        ],
    },
    structure: { score: 0.9, items: [{ severity: "warn", code: "word_count", message: "short" }] },
    links: {
        placed: [{ target: { pageId: "p2", url: "/b/", relation: "up", required: true }, anchor: "b", paragraphIndex: 2 }],
        missing: [{ pageId: "p3", url: "/c/", relation: "down", required: true, anchors: [] }],
        decisions: [],
    },
};

describe("validationView", () => {
    it("splits compliance from structure and keeps both scores", () => {
        const view = validationView(validation);
        expect(view?.complianceScore).toBe(0.75);
        expect(view?.structureScore).toBe(0.9);
        expect(view?.compliance).toHaveLength(3);
        expect(view?.structure).toHaveLength(1);
    });

    it("reads the placements and the misses", () => {
        const view = validationView(validation);
        expect(view?.placed[0].anchor).toBe("b");
        expect(view?.placed[0].target.url).toBe("/b/");
        expect(view?.missing[0].url).toBe("/c/");
    });

    it("answers null for an unreadable payload", () => {
        expect(validationView(null)).toBeNull();
        expect(validationView("<html>")).toBeNull();
    });
});

describe("actualLinks", () => {
    it("classifies placements as graph links and reads the rest off the findings", () => {
        const view = validationView(validation);
        const links = actualLinks(view ?? { ...emptyValidation });
        expect(links).toStrictEqual([
            { href: "/b/", anchor: "b", kind: "graph" },
            { href: "/x/", anchor: "", kind: "unknown_internal" },
            { href: "/a/", anchor: "", kind: "self" },
        ]);
    });

    it("counts each class", () => {
        const view = validationView(validation);
        expect(countByClass(actualLinks(view ?? { ...emptyValidation }))).toStrictEqual({
            graph: 1,
            self: 1,
            external: 0,
            unknown_internal: 1,
        });
    });
});

const emptyValidation = {
    pageId: "",
    score: null,
    complianceScore: null,
    structureScore: null,
    compliance: [],
    structure: [],
    placed: [],
    missing: [],
};

describe("owedTargets", () => {
    it("marks a target satisfied when a placement carries it", () => {
        const context = linkContextView({
            targets: [
                { pageId: "p2", url: "/b/", relation: "up", required: true, anchors: ["b"] },
                { pageId: "p3", url: "/c/", relation: "down", required: true, anchors: [] },
            ],
        });
        const owed = owedTargets(context, validationView(validation));
        expect(owed).toHaveLength(2);
        expect(owed[0].satisfied).toBe(true);
        expect(owed[0].anchor).toBe("b");
        expect(owed[1].satisfied).toBe(false);
    });

    it("falls back to the validation report when the link context is gone", () => {
        const owed = owedTargets(null, validationView(validation));
        expect(owed.map((entry) => entry.target.url).sort()).toStrictEqual(["/b/", "/c/"]);
    });

    it("answers nothing when neither artifact is readable", () => {
        expect(owedTargets(null, null)).toStrictEqual([]);
    });
});

describe("weigh", () => {
    it("counts errors and warnings, never info", () => {
        const view = validationView(validation);
        const findings = [...(view?.compliance ?? []), ...(view?.structure ?? [])];
        expect(weigh(findings)).toStrictEqual({ errors: 2, warnings: 2 });
    });

    it("reads an unknown severity as info", () => {
        const view = validationView({ compliance: { items: [{ severity: "fatal", code: "x", message: "y" }] } });
        expect(view?.compliance[0].severity).toBe("info");
    });
});

describe("judgeView", () => {
    it("reads a score with its issues and suggestions", () => {
        const view = judgeView({ score: 0.82, issues: ["thin"], suggestions: ["add a section", ""] });
        expect(view).toStrictEqual({ score: 0.82, issues: ["thin"], suggestions: ["add a section"] });
    });

    it("answers null rather than a wrong score", () => {
        expect(judgeView({ issues: [] })).toBeNull();
        expect(judgeView({ score: "high" })).toBeNull();
        expect(judgeView(null)).toBeNull();
    });

    it("survives null lists", () => {
        expect(judgeView({ score: 0, issues: null, suggestions: null })).toStrictEqual({
            score: 0,
            issues: [],
            suggestions: [],
        });
    });
});

describe("publishView", () => {
    it("reads the result and its findings", () => {
        const view = publishView({
            url: "https://example.com/a/",
            status: "draft",
            contentHash: "abc",
            wpId: 4790,
            created: true,
            seoApplied: ["title"],
            skipped: ["description"],
            findings: [{ severity: "warn", code: "seo_meta_skipped", message: "no plugin" }],
        });
        expect(view?.wpId).toBe(4790);
        expect(view?.created).toBe(true);
        expect(view?.findings).toHaveLength(1);
    });

    it("leaves a missing id null rather than zero", () => {
        expect(publishView({ url: "" })?.wpId).toBeNull();
    });
});

describe("the remaining artifact shapes", () => {
    it("reads a draft", () => {
        const view = draftView({ title: "T", h1: "H", summary: "S", sections: [{ heading: "A", html: "<p/>" }, 7] });
        expect(view?.sections).toHaveLength(1);
        expect(view?.h1).toBe("H");
    });

    it("reads meta", () => {
        expect(metaView({ title: "T" })?.description).toBe("");
        expect(metaView(7)).toBeNull();
    });

    it("reads images", () => {
        const view = imagesView({ images: [{ role: "hero", url: "u", alt: "a", wpId: 3 }], findings: [], featuredId: 3 });
        expect(view?.images[0].wpId).toBe(3);
        expect(view?.featuredId).toBe(3);
        expect(view?.findings).toStrictEqual([]);
    });

    it("reads what the image step could not do as findings", () => {
        const view = imagesView({
            images: [],
            findings: [
                {
                    severity: "warn",
                    code: "no_image_source",
                    message: "the images of /coffee/espresso/ were not made: no image provider is configured",
                    details: { pageId: "p1", path: "/coffee/espresso/", reason: "no image provider is configured" },
                },
            ],
            featuredId: 0,
        });
        expect(view?.images).toStrictEqual([]);
        expect(view?.findings).toHaveLength(1);
        expect(view?.findings[0].code).toBe("no_image_source");
        expect(view?.findings[0].message).toMatch(/\/coffee\/espresso\//);
    });

    it("reads a relink result under the Go field name", () => {
        const view = relinkView({ linked: 2, conflicts: 0, skipped: 1, neighbors: [{ pageId: "p", path: "/p/" }] });
        expect(view?.neighbours).toHaveLength(1);
        expect(view?.linked).toBe(2);
    });

    it("reads a sync result", () => {
        expect(syncView({ url: "u", status: "publish", links: 7 })?.links).toBe(7);
    });

    it("reads a final report", () => {
        const view = finalView({ pageId: "p", path: "/p/", score: 0.5, errors: 1, warnings: 2, findings: [] });
        expect(view?.errors).toBe(1);
        expect(view?.score).toBe(0.5);
    });
});
