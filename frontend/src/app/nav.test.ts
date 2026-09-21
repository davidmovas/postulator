import { describe, expect, it } from "vitest";

import { goToEntries, railSections } from "./nav.js";

describe("railSections", () => {
    it("opens with the workspace pair and closes with the settings, whatever the route", () => {
        const sections = railSections(null, 0);
        expect(sections.map((section) => section.key)).toEqual(["workspace", "settings"]);
        expect(sections[0].entries.map((entry) => entry.key)).toEqual(["sites", "agent"]);
        expect(sections[0].pinned).toBeUndefined();
        expect(sections[1].entries.map((entry) => entry.key)).toEqual(["settings"]);
        expect(sections[1].pinned).toBe(true);
    });

    it("puts everything a site owns between the pair and the settings", () => {
        const sections = railSections("s1", 0);
        expect(sections.map((section) => section.key)).toEqual(["workspace", "site", "production", "settings"]);
        expect(sections[1].entries.map((entry) => entry.key)).toEqual([
            "overview",
            "graph",
            "pages",
            "links",
            "runs",
        ]);
        expect(sections[2].entries.map((entry) => entry.to)).toEqual([
            "/s/s1/templates",
            "/s/s1/schedules",
            "/s/s1/import",
            "/s/s1/reports",
        ]);
    });

    it("carries the pending count on the agent entry alone", () => {
        const workspace = railSections("s1", 4).find((section) => section.key === "workspace");
        expect(workspace?.entries.map((entry) => entry.badge)).toEqual([undefined, 4]);
    });
});

describe("goToEntries", () => {
    it("is the global entries alone without a site", () => {
        expect(goToEntries(null).map((entry) => entry.key)).toEqual(["sites", "agent", "settings"]);
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
            "sites",
            "agent",
            "settings",
        ]);
    });

    it("carries no badge, because a count is not a destination", () => {
        expect(goToEntries("s1").every((entry) => entry.badge === undefined || entry.badge === 0)).toBe(true);
    });
});
