import { describe, expect, it } from "vitest";

import { findingOptions } from "../../generated/vocab.js";
import { optionDoes, optionLabel } from "./labels.js";

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
