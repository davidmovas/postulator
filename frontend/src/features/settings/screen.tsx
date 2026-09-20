import type { ReactElement } from "react";
import { NavLink, Outlet } from "react-router";

import { copy } from "../../copy/index.js";
import { cx } from "../../ui/index.js";

interface Tab {
    to: string;
    label: string;
}

const tabs: readonly Tab[] = [
    { to: "/settings/general", label: copy.settings.tabs.general },
    { to: "/settings/models", label: copy.settings.tabs.models },
    { to: "/settings/security", label: copy.settings.tabs.security },
    { to: "/settings/about", label: copy.settings.tabs.about },
];

export function SettingsScreen(): ReactElement {
    return (
        <div className="flex h-full min-h-0 flex-col">
            <header className="flex shrink-0 flex-col gap-1 px-4 pt-4 pb-3">
                <h1 className="text-xl font-semibold tracking-tight text-ink">{copy.settings.title}</h1>
                <p className="max-w-2xl text-sm text-ink-soft">{copy.settings.subtitle}</p>
            </header>
            <nav aria-label={copy.nav.settings} className="flex shrink-0 gap-0.5 border-b border-hairline px-3">
                {tabs.map(({ to, label }) => (
                    <NavLink
                        key={to}
                        to={to}
                        className={({ isActive }) =>
                            cx(
                                "inline-flex h-7 items-center px-2.5 text-sm font-medium transition-colors duration-100 ease-out",
                                isActive
                                    ? "font-semibold text-ink shadow-[inset_0_-2px_0_var(--color-accent)]"
                                    : "text-ink-dim hover:text-ink",
                            )
                        }
                    >
                        {label}
                    </NavLink>
                ))}
            </nav>
            <div className="min-h-0 flex-1 overflow-auto p-4">
                <Outlet />
            </div>
        </div>
    );
}
