import { describe, expect, it } from "vitest";

import { goToEntries, railSections } from "./nav.js";

describe("railSections", () => {
    it("shows the global section alone when no site is in the route", () => {
        const sections = railSections(null, 0);
        expect(sections.map((section) => section.key)).toEqual(["global"]);
        expect(sections[0].pinned).toBe(true);
    });

    it("adds the site and production sections when a site is in the route", () => {
        const sections = railSections("s1", 0);
        expect(sections.map((section) => section.key)).toEqual(["site", "production", "global"]);
        expect(sections[0].entries.map((entry) => entry.key)).toEqual([
            "overview",
            "graph",
            "pages",
            "links",
            "runs",
        ]);
        expect(sections[1].entries.map((entry) => entry.to)).toEqual([
            "/s/s1/templates",
            "/s/s1/schedules",
            "/s/s1/import",
            "/s/s1/reports",
        ]);
    });

    it("carries the pending count on the agent entry alone", () => {
        const global = railSections("s1", 4).find((section) => section.key === "global");
        expect(global?.entries.map((entry) => entry.badge)).toEqual([4, undefined, undefined]);
    });
});

describe("goToEntries", () => {
    it("is the global entries alone without a site", () => {
        expect(goToEntries(null).map((entry) => entry.key)).toEqual(["agent", "sites", "settings"]);
    });

    it("is every screen of the site followed by the global ones", () => {
        expect(goToEntries("s1").map((entry) => entry.key)).toEqual([
            "overview",
            "graph",
            "pages",
            "links",
            "runs",
            "templates",
            "schedules",
            "import",
            "reports",
            "agent",
            "sites",
            "settings",
        ]);
    });

    it("carries no badge, because a count is not a destination", () => {
        expect(goToEntries("s1").every((entry) => entry.badge === undefined || entry.badge === 0)).toBe(true);
    });
});
