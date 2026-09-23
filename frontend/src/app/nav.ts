import { copy } from "../copy/index.js";
import { settingsHome } from "../features/settings/index.js";
import type { IconComponent } from "../ui/icons/index.js";
import {
    AccountTreeIcon,
    DashboardCustomizeIcon,
    HubIcon,
    LinkIcon,
    MonitoringIcon,
    PlayCircleIcon,
    PublicIcon,
    ScheduleIcon,
    SettingsIcon,
    SmartToyIcon,
    SpaceDashboardIcon,
    UploadFileIcon,
} from "../ui/icons/index.js";

export interface NavEntry {
    key: string;
    label: string;
    to: string;
    Icon: IconComponent;
    badge?: number;
}

export interface NavSection {
    key: string;
    entries: readonly NavEntry[];
    pinned?: boolean;
}

export function siteEntries(siteId: string): readonly NavEntry[] {
    return [
        { key: "overview", label: copy.nav.overview, to: `/s/${siteId}/overview`, Icon: SpaceDashboardIcon },
        { key: "graph", label: copy.nav.graph, to: `/s/${siteId}/graph`, Icon: HubIcon },
        { key: "pages", label: copy.nav.pages, to: `/s/${siteId}/pages`, Icon: AccountTreeIcon },
        { key: "links", label: copy.nav.links, to: `/s/${siteId}/links`, Icon: LinkIcon },
        { key: "runs", label: copy.nav.runs, to: `/s/${siteId}/runs`, Icon: PlayCircleIcon },
    ];
}

export function productionEntries(siteId: string): readonly NavEntry[] {
    return [
        { key: "templates", label: copy.nav.templates, to: `/s/${siteId}/templates`, Icon: DashboardCustomizeIcon },
        { key: "schedules", label: copy.nav.schedules, to: `/s/${siteId}/schedules`, Icon: ScheduleIcon },
        { key: "import", label: copy.nav.importExport, to: `/s/${siteId}/import`, Icon: UploadFileIcon },
        { key: "reports", label: copy.nav.reports, to: `/s/${siteId}/reports`, Icon: MonitoringIcon },
    ];
}

export function workspaceEntries(pending: number): readonly NavEntry[] {
    return [
        { key: "sites", label: copy.nav.sites, to: "/sites", Icon: PublicIcon },
        { key: "agent", label: copy.nav.agent, to: "/agent", Icon: SmartToyIcon, badge: pending },
    ];
}

export function settingsEntry(): NavEntry {
    return { key: "settings", label: copy.nav.settings, to: settingsHome, Icon: SettingsIcon };
}

export function globalEntries(pending: number): readonly NavEntry[] {
    return [...workspaceEntries(pending), settingsEntry()];
}

export function railSections(siteId: string | null, pending: number): readonly NavSection[] {
    const sections: NavSection[] = [{ key: "workspace", entries: workspaceEntries(pending) }];
    if (siteId !== null) {
        sections.push({ key: "site", entries: siteEntries(siteId) });
        sections.push({ key: "production", entries: productionEntries(siteId) });
    }
    sections.push({ key: "settings", entries: [settingsEntry()], pinned: true });
    return sections;
}

export function goToEntries(siteId: string | null): readonly NavEntry[] {
    const listed = siteId === null ? [] : [...siteEntries(siteId), ...productionEntries(siteId)];
    return [...listed, ...globalEntries(0)];
}
