import type { ReactElement } from "react";
import { Outlet } from "react-router";

import { copy } from "../../copy/index.js";
import { Tabs } from "../../ui/index.js";
import type { TabItem } from "../../ui/index.js";

const tabs: readonly TabItem[] = [
    { key: "general", label: copy.settings.tabs.general, to: "/settings/general" },
    { key: "models", label: copy.settings.tabs.models, to: "/settings/models" },
    { key: "security", label: copy.settings.tabs.security, to: "/settings/security" },
    { key: "about", label: copy.settings.tabs.about, to: "/settings/about" },
];

export function SettingsScreen(): ReactElement {
    return (
        <div className="flex h-full min-h-0 flex-col">
            <header className="flex shrink-0 flex-col gap-1 px-4 pt-4 pb-3">
                <h1 className="text-xl font-semibold tracking-tight text-ink">{copy.settings.title}</h1>
                <p className="max-w-2xl text-sm text-ink-soft">{copy.settings.subtitle}</p>
            </header>
            <div className="flex shrink-0 border-b border-hairline px-3">
                <Tabs label={copy.nav.settings} items={tabs} />
            </div>
            <div className="min-h-0 flex-1 overflow-auto p-4">
                <Outlet />
            </div>
        </div>
    );
}
