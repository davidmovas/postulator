import { describe, expect, it } from "vitest";

import { actors, pageStatuses, publishModes } from "../../generated/vocab.js";
import { actorLabel, pageStatusLabel, publishLabel, targetsSentence } from "./labels.js";

const vocabularies: readonly [string, readonly string[], (value: string) => string][] = [
    ["page status", pageStatuses, pageStatusLabel],
    ["publish mode", publishModes, publishLabel],
    ["actor", actors, actorLabel],
];

describe("every vocabulary reaches the screen as words", () => {
    it.each(vocabularies)("labels every %s", (_name, values, label) => {
        for (const value of values) {
            expect(label(value)).not.toBe("");
            expect(label(value)).not.toContain("_");
        }
    });

    it.each(vocabularies)("shows an unknown %s as it arrived", (_name, _values, label) => {
        expect(label("something_new")).toBe("something_new");
    });
});

describe("targetsSentence", () => {
    it.each([
        ["planned", 5, "", "Up to 5 planned pages"],
        ["", 500, "", "Up to 500 pages"],
        ["", 20, "Kiln Care", "Up to 20 pages of Kiln Care"],
        ["published", 20, "Kiln Care", "Up to 20 published pages of Kiln Care"],
        ["", 0, "", "Up to 500 pages"],
    ])("says %s/%i/%s as %s", (status, limit, entity, want) => {
        expect(targetsSentence(status, limit, entity)).toBe(want);
    });
});
