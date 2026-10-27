import { copy } from "../../copy/index.js";
import type { AdoptionSkip, ProposalClash } from "../../generated/vocab.js";
import { adoptionSkips, isOneOf, proposalClashes } from "../../generated/vocab.js";
import type { Tone } from "../../ui/index.js";
import type { EntityType, NodeState } from "./model/index.js";
import type { Lens } from "./model/lens.js";

type Clash = Exclude<ProposalClash, "">;

const clashKinds: readonly Clash[] = proposalClashes.filter((held): held is Clash => held !== "");

const clashWords: Readonly<Record<Clash, string>> = copy.graph.ai.clashes;

export function clashLabel(clash: string): string {
    return isOneOf(clashKinds, clash) ? clashWords[clash] : clash;
}

const skipWords: Readonly<Record<AdoptionSkip, string>> = copy.graph.adopt.skips;

export function adoptionSkipLabel(skip: string): string {
    return isOneOf(adoptionSkips, skip) ? skipWords[skip] : skip;
}

export function entityTypeLabel(type: EntityType | null): string {
    return type === null ? copy.pageTypes.untyped : type.name;
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
