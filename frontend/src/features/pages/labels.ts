import { copy } from "../../copy/index.js";
import type { PageStatus } from "../../generated/vocab.js";
import { isOneOf, pageStatuses } from "../../generated/vocab.js";
import type { IconComponent, Tone } from "../../ui/index.js";
import { BoltIcon, TravelExploreIcon } from "../../ui/index.js";

const statusTones: Readonly<Record<PageStatus, Tone>> = {
    planned: "info",
    exists: "accent",
    published: "ok",
    archived: "muted",
};

export function pageStatusTone(status: string): Tone {
    return isOneOf(pageStatuses, status) ? statusTones[status] : "muted";
}

export function pageStatusLabel(status: string): string {
    return isOneOf(pageStatuses, status) ? copy.pages.statuses[status] : status;
}

export function linkOwnerTone(ours: boolean): Tone {
    return ours ? "accent" : "muted";
}

export function linkOwnerIcon(ours: boolean): IconComponent {
    return ours ? BoltIcon : TravelExploreIcon;
}

export function linkOwnerLabel(ours: boolean): string {
    return ours ? copy.pages.owners.ours : copy.pages.owners.theirs;
}
