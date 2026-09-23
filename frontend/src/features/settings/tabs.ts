import { copy } from "../../copy/index.js";
import { browserSettingsPath } from "../../data/errors.js";
import type { TabItem } from "../../ui/index.js";
import type { SettingsTabKey } from "./model/layout.js";

export const settingsHome = "/settings/models";

export const settingsTabs: readonly TabItem<SettingsTabKey>[] = [
    { key: "models", label: copy.settings.tabs.models, to: settingsHome },
    { key: "runs", label: copy.settings.tabs.runs, to: "/settings/runs" },
    { key: "agent", label: copy.settings.tabs.agent, to: "/settings/agent" },
    { key: "browser", label: copy.settings.tabs.browser, to: browserSettingsPath },
    { key: "security", label: copy.settings.tabs.security, to: "/settings/security" },
    { key: "about", label: copy.settings.tabs.about, to: "/settings/about" },
];
