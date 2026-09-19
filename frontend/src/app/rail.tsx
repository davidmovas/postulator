import { NavLink } from "react-router";

import { copy } from "../copy/index.js";
import { cx } from "../ui/index.js";
import type { IconComponent } from "../ui/index.js";
import {
    AccountTreeIcon,
    DashboardCustomizeIcon,
    HubIcon,
    MonitoringIcon,
    PlayCircleIcon,
    PublicIcon,
    ScheduleIcon,
    SettingsIcon,
    SmartToyIcon,
    SpaceDashboardIcon,
    UploadFileIcon,
} from "../ui/index.js";

interface RailEntry {
    to: string;
    label: string;
    Icon: IconComponent;
}

function siteEntries(siteId: string): RailEntry[] {
    return [
        { to: `/s/${siteId}/overview`, label: copy.nav.overview, Icon: SpaceDashboardIcon },
        { to: `/s/${siteId}/graph`, label: copy.nav.graph, Icon: HubIcon },
        { to: `/s/${siteId}/pages`, label: copy.nav.pages, Icon: AccountTreeIcon },
        { to: `/s/${siteId}/runs`, label: copy.nav.runs, Icon: PlayCircleIcon },
        { to: `/s/${siteId}/templates`, label: copy.nav.templates, Icon: DashboardCustomizeIcon },
        { to: `/s/${siteId}/schedules`, label: copy.nav.schedules, Icon: ScheduleIcon },
        { to: `/s/${siteId}/import`, label: copy.nav.importExport, Icon: UploadFileIcon },
        { to: `/s/${siteId}/reports`, label: copy.nav.reports, Icon: MonitoringIcon },
    ];
}

const globalEntries: RailEntry[] = [
    { to: "/sites", label: copy.nav.sites, Icon: PublicIcon },
    { to: "/agent", label: copy.nav.agent, Icon: SmartToyIcon },
    { to: "/settings/general", label: copy.nav.settings, Icon: SettingsIcon },
];

export interface RailProps {
    siteId: string | null;
}

export function Rail({ siteId }: RailProps) {
    const entries = siteId === null ? globalEntries : [...siteEntries(siteId), ...globalEntries];

    return (
        <nav
            aria-label={copy.shell.sections}
            className="flex w-12 shrink-0 flex-col items-center gap-0.5 border-r border-hairline bg-panel py-2"
        >
            {entries.map(({ to, label, Icon }) => (
                <NavLink
                    key={to}
                    to={to}
                    title={label}
                    aria-label={label}
                    className={({ isActive }) =>
                        cx(
                            "relative flex h-8 w-8 items-center justify-center rounded-md transition-colors duration-100 ease-out",
                            isActive
                                ? "bg-accent-soft text-accent"
                                : "text-ink-dim hover:bg-inset hover:text-ink-soft",
                        )
                    }
                >
                    {({ isActive }) => (
                        <>
                            {isActive ? (
                                <span
                                    aria-hidden={true}
                                    className="absolute -left-2 h-4 w-0.5 rounded-full bg-accent"
                                />
                            ) : null}
                            <Icon size={20} />
                        </>
                    )}
                </NavLink>
            ))}
        </nav>
    );
}
