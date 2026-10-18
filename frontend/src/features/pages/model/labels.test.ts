import { describe, expect, it } from "vitest";

import {
    contentRootStates,
    findingCodes,
    findingSeverities,
    linkSources,
    linkTypes,
    pageWpTypes,
    parentMethods,
    parentReasons,
    regionKinds,
    technicalReasons,
    trailSources,
    urlForms,
} from "../../../generated/vocab.js";
import {
    findingCodeLabel,
    linkSourceLabel,
    linkTypeLabel,
    parentMethodLabel,
    parentMethodName,
    parentReasonLabel,
    parentReasonSentence,
    parentSourceLabel,
    readOnlyReasonLabel,
    regionKindLabel,
    rootStateLabel,
    rootStateTone,
    severityLabel,
    severityTone,
    technicalReasonLabel,
    trailSourceLabel,
    urlFormLabel,
    wpTypeLabel,
    fieldLabel,
} from "./labels.js";

const vocabularies: readonly [string, readonly string[], (value: string) => string][] = [
    ["link types", linkTypes, linkTypeLabel],
    ["link sources", linkSources, linkSourceLabel],
    ["region kinds", regionKinds, regionKindLabel],
    ["parent reasons", parentReasons, parentReasonLabel],
    ["parent sources", parentReasons, parentSourceLabel],
    ["parent methods", parentMethods, parentMethodLabel],
    ["parent method names", parentMethods, parentMethodName],
    ["trail sources", trailSources, trailSourceLabel],
    ["technical reasons", technicalReasons, technicalReasonLabel],
    ["content root states", contentRootStates, rootStateLabel],
    ["finding codes", findingCodes, findingCodeLabel],
    ["finding severities", findingSeverities, severityLabel],
    ["url forms", urlForms, urlFormLabel],
    ["WordPress types", pageWpTypes, wpTypeLabel],
];

describe("the page model's vocabulary in words", () => {
    it.each(vocabularies)("names every one of the %s without printing the stored value", (_name, values, label) => {
        for (const value of values) {
            const worded = label(value);
            expect(worded).not.toBe("");
            expect(worded).not.toBe(value);
            expect(worded).not.toContain("_");
        }
    });

    it.each(vocabularies)("shows one of the %s it has never heard of as it arrived", (_name, _values, label) => {
        expect(label("teleported")).toBe("teleported");
    });

    it("words a parent's reason with the method that resolved it", () => {
        expect(parentReasonSentence("jsonld_breadcrumb", "href")).toBe("from the JSON-LD breadcrumb, by its link");
        expect(parentReasonSentence("jsonld_breadcrumb", "redirect")).toBe("from the JSON-LD breadcrumb, through a redirect");
        expect(parentReasonSentence("aria_breadcrumb", "name")).toBe("from the breadcrumb navigation, by its name");
        expect(parentReasonSentence("wp_parent", "")).toBe("from the WordPress parent");
        expect(parentReasonSentence("url", "telepathy")).toBe("from the address");
    });

    it("says a page with no internal link has no link form", () => {
        expect(urlFormLabel("")).not.toBe("");
    });

    it("words the read-only reasons", () => {
        expect(readOnlyReasonLabel("term")).toBe("A category page, read only");
        expect(readOnlyReasonLabel("builder:elementor")).toBe("Made with elementor, read only");
        expect(readOnlyReasonLabel("teleported")).toBe("teleported");
    });

    it("words a field key, a keyed one by its base and an unknown one plainly", () => {
        expect(fieldLabel("seo_title")).toBe("SEO title");
        expect(fieldLabel("variation_description:41")).toBe("variation description");
        expect(fieldLabel("mystery_key")).toBe("field");
    });

    it("gives every severity and every root state a tone", () => {
        expect(findingSeverities.map((severity) => severityTone(severity))).toStrictEqual(["info", "warn", "danger"]);
        expect(severityTone("teleported")).toBe("muted");
        expect(rootStateTone("not_found")).toBe("warn");
        expect(rootStateTone("teleported")).toBe("muted");
    });
});
