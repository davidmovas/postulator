import { copy } from "../../copy/index.js";
import type { SiteStatus } from "../../generated/vocab.js";
import { isOneOf, siteStatuses } from "../../generated/vocab.js";
import type { Tone } from "../../ui/index.js";

const tones: Readonly<Record<SiteStatus, Tone>> = {
    active: "ok",
    paused: "muted",
    error: "danger",
};

export function siteStatusTone(status: string): Tone {
    return isOneOf(siteStatuses, status) ? tones[status] : "muted";
}

export function siteStatusLabel(status: string): string {
    return isOneOf(siteStatuses, status) ? copy.sites.statuses[status] : status;
}
