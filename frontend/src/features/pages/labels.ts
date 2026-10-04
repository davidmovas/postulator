import { copy } from "../../copy/index.js";
import type { LinkOrigin, PageStatus } from "../../generated/vocab.js";
import { isOneOf, linkOrigins, pageStatuses } from "../../generated/vocab.js";
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

const originTones: Readonly<Record<LinkOrigin, Tone>> = {
    generated: "accent",
    observed: "muted",
};

const originIcons: Readonly<Record<LinkOrigin, IconComponent>> = {
    generated: BoltIcon,
    observed: TravelExploreIcon,
};

export function originTone(origin: string): Tone {
    return isOneOf(linkOrigins, origin) ? originTones[origin] : "muted";
}

export function originIcon(origin: string): IconComponent {
    return isOneOf(linkOrigins, origin) ? originIcons[origin] : TravelExploreIcon;
}
