import {
    Bot,
    CalendarClock,
    FileStack,
    Gauge,
    Import,
    LayoutTemplate,
    ListChecks,
    Network,
    Settings,
    SquareStack,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { NavLink } from "react-router";

import { copy } from "../copy/index.js";

interface RailEntry {
    to: string;
    label: string;
    Icon: LucideIcon;
}

function siteEntries(siteId: string): RailEntry[] {
    return [
        { to: `/s/${siteId}/overview`, label: copy.nav.overview, Icon: Gauge },
        { to: `/s/${siteId}/graph`, label: copy.nav.graph, Icon: Network },
        { to: `/s/${siteId}/pages`, label: copy.nav.pages, Icon: FileStack },
        { to: `/s/${siteId}/runs`, label: copy.nav.runs, Icon: ListChecks },
        { to: `/s/${siteId}/templates`, label: copy.nav.templates, Icon: LayoutTemplate },
        { to: `/s/${siteId}/schedules`, label: copy.nav.schedules, Icon: CalendarClock },
        { to: `/s/${siteId}/import`, label: copy.nav.importExport, Icon: Import },
        { to: `/s/${siteId}/reports`, label: copy.nav.reports, Icon: SquareStack },
    ];
}

const globalEntries: RailEntry[] = [
    { to: "/sites", label: copy.nav.sites, Icon: SquareStack },
    { to: "/agent", label: copy.nav.agent, Icon: Bot },
    { to: "/settings/general", label: copy.nav.settings, Icon: Settings },
];

export interface RailProps {
    siteId: string | null;
}

export function Rail({ siteId }: RailProps) {
    const entries = siteId === null ? globalEntries : [...siteEntries(siteId), ...globalEntries];

    return (
        <nav className="flex w-12 shrink-0 flex-col items-center gap-1 border-r border-base-700 bg-base-900 py-2">
            {entries.map(({ to, label, Icon }) => (
                <NavLink
                    key={to}
                    to={to}
                    title={label}
                    aria-label={label}
                    className={({ isActive }) =>
                        `flex h-9 w-9 items-center justify-center rounded-panel ${
                            isActive ? "bg-base-700 text-accent-400" : "text-ink-400 hover:bg-base-800"
                        }`
                    }
                >
                    <Icon size={17} strokeWidth={1.75} />
                </NavLink>
            ))}
        </nav>
    );
}
