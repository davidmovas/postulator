import { copy } from "../../copy/index.js";
import { stepNames } from "../../generated/vocab.js";
import type { SpecDraft } from "./spec.js";

const imageStep = "generate_images";

export function blankDraft(): SpecDraft {
    return {
        sections: [
            {
                heading: copy.templates.blank.firstHeading,
                intent: copy.templates.blank.firstIntent,
                targetWords: 200,
                required: true,
                include: [],
                primaryInHeading: true,
            },
        ],
        tone: "",
        lengthMin: 600,
        lengthMax: 1200,
        primaryInTitle: true,
        primaryInH1: true,
        primaryInFirstParagraph: true,
        maxDensity: 0.02,
        upDepth: 1,
        downLinks: true,
        siblingMinWeight: 0.5,
        maxLinks: 8,
        maxPerTarget: 1,
        parentLinkWithinParagraphs: 2,
        childrenSection: false,
        titlePattern: "{primaryKeyword} | {siteName}",
        descriptionMax: 155,
        featuredImage: false,
        inlineImages: 0,
        imageSource: "wpmedia",
        profiles: [],
        recipe: stepNames.map((name) => ({ name, enabled: name !== imageStep, declared: true, params: null })),
    };
}
