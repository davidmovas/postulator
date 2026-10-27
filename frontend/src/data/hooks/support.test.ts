import { describe, expect, it, vi } from "vitest";

vi.mock("../host.js", () => ({ pickSaveFile: () => Promise.resolve(null) }));

const { bundleName, bundleRefusal } = await import("./support.js");

describe("the support bundle's file", () => {
    const at = new Date("2026-10-27T09:30:00Z");

    it.each([
        ["Clay and Kiln", "support-bundle-clay-and-kiln-2026-10-27.json"],
        ["  Café / Shop!  ", "support-bundle-caf-shop-2026-10-27.json"],
        ["", "support-bundle-2026-10-27.json"],
    ])("is named after the site %j and the day", (site, name) => {
        expect(bundleName(site, at)).toBe(name);
    });

    it("says why it could not be written in the words the app was given", () => {
        const thrown = Object.assign(new Error("refused"), {
            cause: { code: "EXTERNAL", message: "the support bundle could not be written to C:\\x.json" },
        });
        expect(bundleRefusal(thrown)).toBe("the support bundle could not be written to C:\\x.json");
        expect(bundleRefusal(null)).toBeNull();
    });
});
