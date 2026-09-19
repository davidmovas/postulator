import type { EntityKind, ItemStatus, LinkOrigin, PageStatus } from "../../generated/vocab.js";
import { entityKinds, isOneOf, itemStatuses, linkOrigins, pageStatuses } from "../../generated/vocab.js";
import type { IconComponent, Tone } from "../../ui/index.js";
import {
    BoltIcon,
    CategoryIcon,
    HubIcon,
    Inventory2Icon,
    StarShineIcon,
    TopicIcon,
    TravelExploreIcon,
} from "../../ui/index.js";

const statusTones: Readonly<Record<PageStatus, Tone>> = {
    planned: "info",
    exists: "accent",
    published: "ok",
    archived: "muted",
};

export function statusTone(status: string): Tone {
    return isOneOf(pageStatuses, status) ? statusTones[status] : "muted";
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

const entityIcons: Readonly<Record<EntityKind, IconComponent>> = {
    hub: HubIcon,
    product: Inventory2Icon,
    topic: TopicIcon,
    category: CategoryIcon,
    custom: StarShineIcon,
};

export function entityIcon(kind: string): IconComponent {
    return isOneOf(entityKinds, kind) ? entityIcons[kind] : StarShineIcon;
}

const itemStatusTones: Readonly<Record<ItemStatus, Tone>> = {
    pending: "muted",
    running: "info",
    waiting: "info",
    paused: "warn",
    completed: "ok",
    failed: "danger",
    cancelled: "muted",
};

export function itemStatusTone(status: string): Tone {
    return isOneOf(itemStatuses, status) ? itemStatusTones[status] : "muted";
}
