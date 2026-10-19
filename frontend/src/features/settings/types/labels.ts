import { copy } from "../../../copy/index.js";
import type { PageTypePreset } from "../../../generated/vocab.js";
import {
    anchorStrategies,
    fieldKeys,
    fieldProviders,
    isOneOf,
    pageTypePresets,
    pageTypeReasons,
    pageTypeSources,
} from "../../../generated/vocab.js";
import type { IconComponent, Tone } from "../../../ui/index.js";
import {
    BuildIcon,
    CategoryIcon,
    CropSquareIcon,
    HubIcon,
    Inventory2Icon,
    TableChartIcon,
    TopicIcon,
} from "../../../ui/index.js";

export const untyped = "untyped";

export const presetRoles = [...pageTypePresets, untyped] as const;

export type PresetRole = (typeof presetRoles)[number];

const presetIcons: Readonly<Record<PresetRole, IconComponent>> = {
    hub: HubIcon,
    entity: TopicIcon,
    product: Inventory2Icon,
    store_category: CategoryIcon,
    catalog: TableChartIcon,
    technical: BuildIcon,
    untyped: CropSquareIcon,
};

const presetTones: Readonly<Record<PresetRole, Tone>> = {
    hub: "accent",
    entity: "muted",
    product: "ok",
    store_category: "info",
    catalog: "info",
    technical: "warn",
    untyped: "muted",
};

function spaced(value: string): string {
    return value.replaceAll("_", " ");
}

export function roleOf(preset: string | null | undefined): PresetRole {
    return preset !== null && preset !== undefined && isOneOf(pageTypePresets, preset) ? preset : untyped;
}

export function presetLabel(preset: string): string {
    if (preset === "" || preset === untyped) {
        return copy.pageTypes.untyped;
    }
    return isOneOf(pageTypePresets, preset) ? copy.pageTypes.presets[preset] : spaced(preset);
}

export function presetHint(preset: string): string {
    return isOneOf(pageTypePresets, preset) ? copy.pageTypes.presetHints[preset] : "";
}

export function presetIcon(preset: string): IconComponent {
    return presetIcons[roleOf(preset)];
}

export function presetTone(preset: string): Tone {
    return presetTones[roleOf(preset)];
}

export function isPreset(value: string): value is PageTypePreset {
    return isOneOf(pageTypePresets, value);
}

export function sourceLabel(source: string): string {
    return isOneOf(pageTypeSources, source) ? copy.pageTypes.sources[source] : spaced(source);
}

export function reasonLabel(reason: string): string {
    return isOneOf(pageTypeReasons, reason) ? copy.pageTypes.reasons[reason] : spaced(reason);
}

export const metaPrefix = "meta:";

export function metaKeyOf(field: string): string | null {
    return field.startsWith(metaPrefix) ? field.slice(metaPrefix.length) : null;
}

export function fieldLabel(field: string, label = ""): string {
    const meta = metaKeyOf(field);
    if (meta !== null) {
        return label.trim() === "" ? meta : label;
    }
    return isOneOf(fieldKeys, field) ? copy.pageTypes.fields[field] : spaced(field);
}

export function providerLabel(provider: string): string {
    return isOneOf(fieldProviders, provider) ? copy.pageTypes.providers[provider] : spaced(provider);
}

export function anchorLabel(strategy: string): string {
    return isOneOf(anchorStrategies, strategy) ? copy.pageTypes.anchors[strategy] : spaced(strategy);
}

export function decimal(value: number): string {
    return Number.isInteger(value) ? value.toFixed(1) : String(value);
}
