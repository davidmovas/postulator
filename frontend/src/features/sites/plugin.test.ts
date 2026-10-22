import { describe, expect, it } from "vitest";

import { copy } from "../../copy/index.js";
import { capabilityLabel, pluginBanner, seoPluginLabel, storeBanner } from "./plugin.js";

const current = {
    installed: true,
    outdated: false,
    version: "1.4.0",
    requiredVersion: "1.4.0",
    capabilities: ["bulk", "raw"],
    seoPlugin: "rankmath",
};

describe("what the panel says about the companion plugin", () => {
    it("names a current plugin's version, capabilities and SEO plugin, and offers no zip", () => {
        const banner = pluginBanner(current, "Shop");
        expect(banner.tone).toBe("ok");
        expect(banner.title).toContain("1.4.0");
        expect(banner.body).toContain("bulk writes");
        expect(banner.body).toContain("Rank Math");
        expect(banner.offersZip).toBe(false);
    });

    it("names an outdated plugin, the version it needs and how to replace it, and offers the zip", () => {
        const banner = pluginBanner({ ...current, outdated: true, version: "1.2.0" }, "Shop");
        expect(banner.tone).toBe("warn");
        expect(banner.title).toContain("1.2.0");
        expect(banner.title).toContain("1.4.0");
        expect(banner.body).toContain("Replace current with uploaded");
        expect(banner.offersZip).toBe(true);
    });

    it("says a missing plugin is missing on the site it names, and offers the zip", () => {
        const banner = pluginBanner({ ...current, installed: false, capabilities: [] }, "Shop");
        expect(banner.tone).toBe("warn");
        expect(banner.title).toContain("Shop");
        expect(banner.offersZip).toBe(true);
    });
});

describe("what the companion plugin reports, in words", () => {
    it("names every capability the plugin manifest can carry", () => {
        const carried = [
            "bulk",
            "seo_meta",
            "seo_meta_read",
            "content_hash",
            "raw",
            "preview",
            "rendered",
            "meta",
            "terms",
            "settings",
            "meta_write",
            "guarded_write",
        ];
        expect(new Set(carried.map(capabilityLabel)).size).toBe(carried.length);
        for (const capability of carried) {
            const label = capabilityLabel(capability);
            expect(label).not.toBe("");
            expect(label).not.toContain("_");
            expect(label).not.toBe(capability);
        }
    });

    it("names no plugin version of its own where a page preview needs a newer plugin", () => {
        expect(copy.pages.preview.outdatedPluginBody).not.toMatch(/\d+\.\d+/);
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
