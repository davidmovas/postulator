import { copy } from "../../copy/index.js";
import type { EntityKind } from "../../generated/vocab.js";
import { entityKinds, isOneOf } from "../../generated/vocab.js";
import type { Tone } from "../../ui/index.js";
import { entityIcon } from "../pages/labels.js";
import type { NodeState } from "./model/index.js";
import type { Lens } from "./model/lens.js";

export { entityIcon };

const kindTones: Readonly<Record<EntityKind, Tone>> = {
    hub: "accent",
    category: "info",
    product: "ok",
    topic: "muted",
    custom: "warn",
};

export function kindTone(kind: string): Tone {
    return isOneOf(entityKinds, kind) ? kindTones[kind] : "muted";
}

const lensTones: Readonly<Record<Lens, Tone>> = {
    all: "muted",
    noPage: "danger",
    proposed: "info",
    orphan: "warn",
    ai: "accent",
};

export function lensTone(lens: Lens): Tone {
    return lensTones[lens];
}

export function lensLabel(lens: Lens): string {
    return copy.graph.lens[lens];
}

export function lensHint(lens: Lens): string {
    return copy.graph.lens.hint[lens];
}

export function kindLabel(kind: string): string {
    return isOneOf(entityKinds, kind) ? copy.graph.kinds[kind] : kind;
}

export function formatScore(score: number): string {
    return score === 0 ? copy.graph.score.none : score.toFixed(2);
}

export function scored(entities: readonly { score: number }[]): boolean {
    return entities.some((entity) => entity.score > 0);
}

const stateTones: Readonly<Record<NodeState, Tone>> = {
    mismatch: "danger",
    working: "accent",
    published: "ok",
    exists: "warn",
    planned: "info",
    archived: "muted",
    noPage: "muted",
};

export function stateTone(state: NodeState): Tone {
    return stateTones[state];
}

export function stateLabel(state: NodeState): string {
    return copy.graph.legend.stateShort[state];
}

export function stateHint(state: NodeState): string {
    return copy.graph.legend.state[state];
}
