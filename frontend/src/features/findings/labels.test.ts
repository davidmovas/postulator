import { describe, expect, it } from "vitest";

import { copy } from "../../copy/index.js";
import { findingOptions, findingStatuses } from "../../generated/vocab.js";
import { optionDoes, optionLabel, resolutionLabel, resolvedByLabel, statusLabel } from "./labels.js";

const runOptions = new Set(["rewrite_field", "relink_page", "rescan"]);

describe("the words of a finding's options", () => {
    it.each(findingOptions.map((option) => [option]))("%s has a word and one sentence", (option) => {
        expect(optionLabel(option)).not.toBe(option);
        const said = optionDoes(option);
        expect(said).toMatch(/^[A-Z][^.]*\.$/);
    });

    it.each(findingOptions.map((option) => [option]))("%s says whether it opens a run or changes the model alone", (option) => {
        if (runOptions.has(option)) {
            expect(optionDoes(option)).toMatch(/^Opens /);
        } else {
            expect(optionDoes(option)).toMatch(/nothing is written to the site\.$/);
        }
    });

    it("keeps an option this build does not know as it came", () => {
        expect(optionLabel("teleport")).toBe("teleport");
        expect(optionDoes("teleport")).toBe("");
    });
});

describe("who settled a finding and how", () => {
    it.each([
        ["person", "a person"],
        ["agent", "the agent"],
        ["observation", "the observation"],
        ["import", "an import"],
        ["", copy.findings.someone],
    ])("names %s", (by, said) => {
        expect(resolvedByLabel(by)).toBe(said);
    });

    it("words every option and every settlement the site records on its own", () => {
        for (const resolution of [...findingOptions, "observed_clear", "row_matched", "row_left_sheet"]) {
            expect(resolutionLabel(resolution)).not.toBe(copy.findings.unknownResolution);
            expect(resolutionLabel(resolution)).not.toContain("_");
        }
        expect(resolutionLabel("teleported")).toBe(copy.findings.unknownResolution);
    });

    it("has a word for every status", () => {
        for (const status of findingStatuses) {
            expect(statusLabel(status)).not.toBe(status);
        }
    });
});
