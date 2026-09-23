import { describe, expect, it } from "vitest";

import { capabilityLabel, seoPluginLabel } from "./plugin.js";

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
