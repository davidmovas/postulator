import { describe, expect, it } from "vitest";

import { pageStatuses } from "../../generated/vocab.js";
import { linkOwnerLabel, linkOwnerTone, pageStatusLabel, pageStatusTone } from "./labels.js";

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

    it("tells our links from the client's by their mark alone", () => {
        expect(linkOwnerLabel(true)).not.toBe("");
        expect(linkOwnerLabel(false)).not.toBe("");
        expect(linkOwnerLabel(true)).not.toBe(linkOwnerLabel(false));
        expect([linkOwnerTone(true), linkOwnerTone(false)]).toStrictEqual(["accent", "muted"]);
    });

    it("gives every page status a tone and an unknown one a muted tone", () => {
        expect(pageStatuses.map((status) => pageStatusTone(status))).toStrictEqual(["info", "accent", "ok", "muted"]);
        expect(pageStatusTone("teleported")).toBe("muted");
    });
});
