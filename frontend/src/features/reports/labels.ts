import { copy } from "../../copy/index.js";
import { entityKinds, isOneOf, pageStatuses } from "../../generated/vocab.js";
import type { Tone } from "../../ui/index.js";
import { noPage } from "./model/site.js";
import type { CoverageReason } from "./model/site.js";

export function reasonLabel(reason: CoverageReason): string {
    if (reason === noPage) {
        return copy.reports.coverage.reasons.noPage;
    }
    return isOneOf(pageStatuses, reason) ? copy.reports.coverage.reasons[reason] : reason;
}

const reasonTones: Readonly<Record<string, Tone>> = {
    noPage: "danger",
    planned: "warn",
    exists: "info",
    archived: "muted",
    published: "ok",
};

export function reasonTone(reason: CoverageReason): Tone {
    return reasonTones[reason] ?? "muted";
}

export function entityKindLabel(kind: string): string {
    return isOneOf(entityKinds, kind) ? copy.imports.entityKinds[kind] : kind;
}

export function shareOf(fraction: number): string {
    return `${Math.round(fraction * 100)}%`;
}

export function shareTone(fraction: number): Tone {
    if (fraction >= 0.9) {
        return "ok";
    }
    return fraction >= 0.6 ? "warn" : "danger";
}
