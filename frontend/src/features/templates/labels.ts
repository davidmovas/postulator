import { copy } from "../../copy/index.js";
import type { Tone } from "../../ui/index.js";

const stepLabels: Readonly<Record<string, string>> = copy.templates.steps;
const roleLabels: Readonly<Record<string, string>> = copy.templates.roles;
const imageSourceLabels: Readonly<Record<string, string>> = copy.templates.imageSource;
const scopeLabels: Readonly<Record<string, string>> = copy.templates.scope;
const overrideScopeLabels: Readonly<Record<string, string>> = {
    site: copy.templates.layer.site,
    page: copy.templates.layer.page,
};

function spaced(value: string): string {
    return value.replaceAll("_", " ");
}

export function stepLabel(name: string): string {
    return stepLabels[name] ?? spaced(name);
}

export function isKnownStep(name: string): boolean {
    return stepLabels[name] !== undefined;
}

export function roleLabel(role: string): string {
    return roleLabels[role] ?? spaced(role);
}

export function imageSourceLabel(source: string): string {
    return source === "" ? copy.templates.meta.noSource : (imageSourceLabels[source] ?? spaced(source));
}

export function scopeLabel(scope: string): string {
    return scopeLabels[scope] ?? spaced(scope);
}

export function scopeTone(scope: string): Tone {
    return scope === "site" ? "accent" : "muted";
}

export function overrideScopeLabel(scope: string): string {
    return overrideScopeLabels[scope] ?? spaced(scope);
}

export function percent(value: number): string {
    const scaled = value * 100;
    const rendered = Number.isInteger(scaled) ? String(scaled) : scaled.toFixed(1);
    return `${rendered}%`;
}

export function flagLabel(on: boolean): string {
    return on ? copy.templates.layer.on : copy.templates.layer.off;
}
