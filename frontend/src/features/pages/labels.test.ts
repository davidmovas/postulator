import { describe, expect, it } from "vitest";

import { entityKinds, pageStatuses } from "../../generated/vocab.js";
import { kindLabel } from "../graph/labels.js";
import { pageStatusLabel } from "./labels.js";

describe("the page vocabulary in words", () => {
    it("names every page status without printing the stored value", () => {
        for (const status of pageStatuses) {
            const label = pageStatusLabel(status);
            expect(label).not.toBe("");
            expect(label).not.toBe(status);
            expect(label).not.toContain("_");
        }
    });

    it("shows a status it has never heard of as it arrived", () => {
        expect(pageStatusLabel("teleported")).toBe("teleported");
    });

    it("names every entity kind from one source", () => {
        for (const kind of entityKinds) {
            expect(kindLabel(kind)).not.toBe("");
            expect(kindLabel(kind)).not.toBe(kind);
        }
    });
});
