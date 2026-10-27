import { describe, expect, it } from "vitest";

import { pageStatuses } from "../../generated/vocab.js";
import { linkOwnerLabel, linkOwnerTone, observedH1Line, pageStatusLabel, pageStatusTone, titleBesideH1 } from "./labels.js";

describe("the H1 a visitor sees", () => {
    it("names the H1 and who prints it", () => {
        expect(observedH1Line({ observedH1: "TB-500", h1Source: "theme" })).toBe(
            "The site shows the H1 “TB-500”, printed by the theme",
        );
        expect(observedH1Line({ observedH1: "BPC-157 explained", h1Source: "content" })).toBe(
            "The site shows the H1 “BPC-157 explained”, from the page’s content",
        );
        expect(observedH1Line({ observedH1: "", h1Source: "none" })).toBe("The site shows no H1");
    });

    it("adds the H1 to a title only where the two differ", () => {
        expect(titleBesideH1({ title: "TB-500", observedH1: "TB-500" })).toBeNull();
        expect(titleBesideH1({ title: "TB-500", observedH1: "" })).toBeNull();
        expect(titleBesideH1({ title: "TB-500", observedH1: "TB-500 for research" })).toBe("TB-500 for research");
    });
});

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
