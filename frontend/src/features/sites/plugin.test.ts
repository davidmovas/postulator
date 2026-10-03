import { describe, expect, it } from "vitest";

import { capabilityLabel, seoPluginLabel, storeBanner } from "./plugin.js";

describe("what the companion plugin reports, in words", () => {
    it("names every capability the plugin manifest can carry", () => {
        for (const capability of ["bulk", "seo_meta", "content_hash", "raw", "preview"]) {
            const label = capabilityLabel(capability);
            expect(label).not.toBe("");
            expect(label).not.toContain("_");
            expect(label).not.toBe(capability);
        }
    });

    it("shows a capability it has never heard of without its underscores", () => {
        expect(capabilityLabel("time_travel")).toBe("time travel");
    });

    it("names the SEO plugins the manifest can report", () => {
        expect(seoPluginLabel("yoast")).toBe("Yoast SEO");
        expect(seoPluginLabel("rankmath")).toBe("Rank Math");
        expect(seoPluginLabel("seopress")).toBe("seopress");
    });
});

describe("what the store lets Postulator do", () => {
    it.each([
        { commerce: "ready", installed: true, tone: "ok" },
        { commerce: "ready", installed: false, tone: "warn" },
        { commerce: "forbidden", installed: true, tone: "warn" },
    ])("speaks of a $commerce store with the plugin installed: $installed", ({ commerce, installed, tone }) => {
        const banner = storeBanner(commerce, installed, "editor");
        expect(banner?.tone).toBe(tone);
        expect(banner?.title).not.toBe("");
        expect(banner?.body).not.toBe("");
    });

    it("says what a ready store never has touched", () => {
        expect(storeBanner("ready", true, "editor")?.body).toMatch(/price/);
    });

    it("names the user a store refuses", () => {
        expect(storeBanner("forbidden", true, "editor")?.body).toContain("editor");
    });

    it.each(["", "absent", "unheard"])("stays quiet about a store that is %s", (commerce) => {
        expect(storeBanner(commerce, true, "editor")).toBeNull();
    });
});
