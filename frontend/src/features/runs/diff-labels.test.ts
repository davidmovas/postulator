import { describe, expect, it } from "vitest";

import { copy } from "../../copy/index.js";
import { blockChanges, blockOwners } from "../../generated/vocab.js";
import { blockState, blockTone, revertedFieldLabel, sentFields } from "./diff-labels.js";

const said = copy.runs.review.diff;

describe("a block's owner and change, in words", () => {
    it("names every owner and change the vocabulary holds without its stored value", () => {
        for (const owner of blockOwners) {
            for (const change of blockChanges) {
                const words = blockState(change, owner);
                expect(words).not.toContain("_");
                expect(words).not.toBe(`${change}: ${owner}`);
            }
        }
    });

    it("reads a kept block of the client's as kept and the client's", () => {
        expect(blockState("kept", "client")).toBe("Kept: the client's");
        expect(blockState("added", "ours")).toBe("Added: ours");
        expect(blockState("added", "site")).toBe("Added: the site's");
    });

    it("shows a change or an owner this build does not know as it came", () => {
        expect(blockState("moved", "robot")).toBe(said.state("moved", "robot"));
    });

    it("tones a removed block as a loss, the site's addition apart from ours, and a kept one quietly", () => {
        expect(blockTone("removed", "ours")).toBe("danger");
        expect(blockTone("added", "ours")).toBe("ok");
        expect(blockTone("added", "site")).toBe("info");
        expect(blockTone("kept", "client")).toBe("muted");
    });
});

describe("what a revert did to a field, in words", () => {
    it("names a put-back field and a handed-back one", () => {
        expect(revertedFieldLabel("restored")).toBe(copy.runs.review.revert.fieldOutcomes["restored"]);
        expect(revertedFieldLabel("kept")).toBe("Handed back");
        expect(revertedFieldLabel("lost")).toBe("lost");
    });
});

describe("the keys a person sends with an acceptance or a revert", () => {
    it("sends nothing to narrow when every key is ticked", () => {
        expect(sentFields(["excerpt", "body"], new Set())).toStrictEqual({ kind: "all" });
        expect(sentFields(["excerpt", "body"], new Set(["seo_title"]))).toStrictEqual({ kind: "all" });
    });

    it("names the ticked keys when some are unticked", () => {
        expect(sentFields(["excerpt", "seo_title", "body"], new Set(["seo_title"]))).toStrictEqual({
            kind: "some",
            fields: ["excerpt", "body"],
        });
    });

    it("never sends an empty list", () => {
        expect(sentFields(["excerpt", "body"], new Set(["excerpt", "body"]))).toStrictEqual({ kind: "none" });
        expect(sentFields([], new Set())).toStrictEqual({ kind: "all" });
    });
});
