import { describe, expect, it } from "vitest";

import { browserSettingsPath } from "../../data/errors.js";
import { settingsTabKeys } from "./model/layout.js";
import { settingsHome, settingsTabs } from "./tabs.js";

describe("the settings tabs", () => {
    it("routes every tab key once, in the declared order", () => {
        expect(settingsTabs.map((tab) => tab.key)).toEqual([...settingsTabKeys]);
    });

    it("gives every tab a route of its own", () => {
        const routes = settingsTabs.map((tab) => tab.to);
        expect(new Set(routes).size).toBe(routes.length);
        expect(routes.every((route) => route?.startsWith("/settings/"))).toBe(true);
    });

    it("opens on the first tab", () => {
        expect(settingsHome).toBe(settingsTabs[0]?.to);
    });

    it("sends a missing Tor Browser to the browser tab", () => {
        expect(browserSettingsPath).toBe(settingsTabs.find((tab) => tab.key === "browser")?.to);
    });
});
