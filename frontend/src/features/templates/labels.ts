import { copy } from "../../copy/index.js";
import type { Tone } from "../../ui/index.js";

const stepLabels: Readonly<Record<string, string>> = copy.templates.steps;
const imageSourceLabels: Readonly<Record<string, string>> = copy.templates.imageSource;
const anchorLabels: Readonly<Record<string, string>> = copy.policies.anchor;
const scopeLabels: Readonly<Record<string, string>> = copy.templates.scope;

export function stepLabel(name: string): string {
    return stepLabels[name] ?? name.replaceAll("_", " ");
}

export function isKnownStep(name: string): boolean {
    return stepLabels[name] !== undefined;
}

export function imageSourceLabel(source: string): string {
    return source === "" ? copy.templates.meta.noSource : (imageSourceLabels[source] ?? source);
}

export function anchorLabel(strategy: string): string {
    return anchorLabels[strategy] ?? strategy;
}

export function scopeLabel(scope: string): string {
    return scopeLabels[scope] ?? scope;
}

export function scopeTone(scope: string): Tone {
    return scope === "site" ? "accent" : "muted";
}

export function decimal(value: number): string {
    return Number.isInteger(value) ? value.toFixed(1) : String(value);
}

export function percent(value: number): string {
    const scaled = value * 100;
    const rendered = Number.isInteger(scaled) ? String(scaled) : scaled.toFixed(1);
    return `${rendered}%`;
}

export function forbidsLabel(external: boolean, self: boolean): string {
    if (external && self) {
        return copy.policies.forbidsBoth;
    }
    if (external) {
        return copy.policies.forbidsExternal;
    }
    if (self) {
        return copy.policies.forbidsSelf;
    }
    return copy.policies.forbidsNone;
}

export function flagLabel(on: boolean): string {
    return on ? copy.templates.layer.on : copy.templates.layer.off;
}
