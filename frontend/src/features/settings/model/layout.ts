import type { SettingGroup } from "../../../generated/vocab.js";

export const settingsTabKeys = ["models", "runs", "agent", "browser", "security", "about"] as const;
export type SettingsTabKey = (typeof settingsTabKeys)[number];

export type SectionId =
    | "modelCalls"
    | "endpoints"
    | "images"
    | "pace"
    | "limits"
    | "history"
    | "cleanup"
    | "wordpress"
    | "importing"
    | "schedules"
    | "other"
    | "agentLimits"
    | "agentDepth"
    | "tor"
    | "browserOther";

export type UnitId = "pages" | "days" | "times" | "perSecond" | "steps" | "rows" | "characters" | "bytes";

export interface SectionSpec {
    id: SectionId;
    tab: SettingsTabKey;
    advanced: boolean;
}

export interface Placement {
    key: string;
    section: SectionId;
    unit?: UnitId;
}

export interface SectionView {
    id: SectionId;
    advanced: boolean;
    keys: readonly string[];
}

export interface TabView {
    plain: readonly SectionView[];
    advanced: readonly SectionView[];
}

const specs: readonly SectionSpec[] = [
    { id: "modelCalls", tab: "models", advanced: true },
    { id: "endpoints", tab: "models", advanced: true },
    { id: "images", tab: "models", advanced: true },
    { id: "pace", tab: "runs", advanced: false },
    { id: "limits", tab: "runs", advanced: false },
    { id: "history", tab: "runs", advanced: false },
    { id: "cleanup", tab: "runs", advanced: true },
    { id: "wordpress", tab: "runs", advanced: true },
    { id: "importing", tab: "runs", advanced: true },
    { id: "schedules", tab: "runs", advanced: true },
    { id: "other", tab: "runs", advanced: true },
    { id: "agentLimits", tab: "agent", advanced: false },
    { id: "agentDepth", tab: "agent", advanced: true },
    { id: "tor", tab: "browser", advanced: false },
    { id: "browserOther", tab: "browser", advanced: true },
];

export const placements: readonly Placement[] = [
    { key: "llm.timeout", section: "modelCalls" },
    { key: "llm.retries", section: "modelCalls", unit: "times" },
    { key: "llm.recordReplayMode", section: "modelCalls" },
    { key: "llm.openai.baseUrl", section: "endpoints" },
    { key: "llm.anthropic.baseUrl", section: "endpoints" },
    { key: "llm.geminiOpenai.baseUrl", section: "endpoints" },
    { key: "llm.gemini.projectId", section: "endpoints" },
    { key: "llm.gemini.location", section: "endpoints" },
    { key: "images.openaiModel", section: "images" },
    { key: "images.localDir", section: "images" },
    { key: "runs.workers", section: "pace", unit: "pages" },
    { key: "runs.perSite", section: "pace", unit: "pages" },
    { key: "wp.rateLimitPerSecond", section: "pace", unit: "perSecond" },
    { key: "runs.deadline", section: "limits" },
    { key: "runs.stepTimeout", section: "limits" },
    { key: "runs.artifactRetentionDays", section: "history", unit: "days" },
    { key: "runs.eventRetentionDays", section: "history", unit: "days" },
    { key: "runs.sweepInterval", section: "cleanup" },
    { key: "runs.leaseDuration", section: "cleanup" },
    { key: "wp.timeout", section: "wordpress" },
    { key: "wp.retries", section: "wordpress", unit: "times" },
    { key: "wp.proxyUrl", section: "wordpress" },
    { key: "sync.batchSize", section: "wordpress", unit: "pages" },
    { key: "import.maxRows", section: "importing", unit: "rows" },
    { key: "schedules.tickInterval", section: "schedules" },
    { key: "agent.turnTimeout", section: "agentLimits" },
    { key: "agent.loopLimit", section: "agentLimits", unit: "steps" },
    { key: "agent.historyBudgetChars", section: "agentDepth", unit: "characters" },
    { key: "agent.maxToolResultBytes", section: "agentDepth", unit: "bytes" },
    { key: "agent.historyToolResultBytes", section: "agentDepth", unit: "bytes" },
    { key: "browser.torPath", section: "tor" },
];

export const groupFallback: Readonly<Record<SettingGroup, SectionSpec>> = {
    agent: { id: "agentDepth", tab: "agent", advanced: true },
    browser: { id: "browserOther", tab: "browser", advanced: true },
    images: { id: "images", tab: "models", advanced: true },
    import: { id: "importing", tab: "runs", advanced: true },
    llm: { id: "modelCalls", tab: "models", advanced: true },
    runs: { id: "cleanup", tab: "runs", advanced: true },
    schedules: { id: "schedules", tab: "runs", advanced: true },
    sync: { id: "wordpress", tab: "runs", advanced: true },
    wp: { id: "wordpress", tab: "runs", advanced: true },
};

const lastResort: SectionSpec = { id: "other", tab: "runs", advanced: true };

const byKey = new Map(placements.map((placement, order) => [placement.key, { placement, order }]));

export function sectionsOf(tab: SettingsTabKey): readonly SectionSpec[] {
    return specs.filter((spec) => spec.tab === tab);
}

export function unitOf(key: string): UnitId | null {
    return byKey.get(key)?.placement.unit ?? null;
}

export function tabOf(key: string): SettingsTabKey {
    return sectionFor(key).tab;
}

export function humanLabel(key: string): string {
    const leaf = key.slice(key.lastIndexOf(".") + 1);
    const words = leaf.replace(/([a-z0-9])([A-Z])/g, "$1 $2").toLowerCase();
    return words.charAt(0).toUpperCase() + words.slice(1);
}

function sectionFor(key: string): { id: SectionId; tab: SettingsTabKey; order: number } {
    const held = byKey.get(key);
    if (held !== undefined) {
        const spec = specs.find((entry) => entry.id === held.placement.section) ?? lastResort;
        return { id: spec.id, tab: spec.tab, order: held.order };
    }
    const group = key.slice(0, Math.max(key.indexOf("."), 0)) as SettingGroup;
    const spec = groupFallback[group] ?? lastResort;
    return { id: spec.id, tab: spec.tab, order: placements.length };
}

export function tabLayout(tab: SettingsTabKey, declared: readonly string[]): TabView {
    const buckets = new Map<SectionId, { key: string; order: number; arrival: number }[]>();

    declared.forEach((key, arrival) => {
        const found = sectionFor(key);
        if (found.tab !== tab) {
            return;
        }
        const held = buckets.get(found.id) ?? [];
        held.push({ key, order: found.order, arrival });
        buckets.set(found.id, held);
    });

    const views = sectionsOf(tab)
        .map((spec) => ({
            id: spec.id,
            advanced: spec.advanced,
            keys: (buckets.get(spec.id) ?? [])
                .sort((left, right) => left.order - right.order || left.arrival - right.arrival)
                .map((entry) => entry.key),
        }))
        .filter((view) => view.keys.length > 0);

    return {
        plain: views.filter((view) => !view.advanced),
        advanced: views.filter((view) => view.advanced),
    };
}
