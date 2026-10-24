import { copy } from "../../copy/index.js";
import type { BlockChange, BlockOwner } from "../../generated/vocab.js";
import { blockChanges, blockOwners, isOneOf } from "../../generated/vocab.js";
import type { Tone } from "../../ui/index.js";

const said = copy.runs.review.diff;

const ownerWords: Readonly<Record<BlockOwner, string>> = said.owners;
const changeWords: Readonly<Record<BlockChange, string>> = said.changes;

export function blockOwnerLabel(owner: string): string {
    return isOneOf(blockOwners, owner) ? ownerWords[owner] : owner;
}

export function blockChangeLabel(change: string): string {
    return isOneOf(blockChanges, change) ? changeWords[change] : change;
}

export function blockState(change: string, owner: string): string {
    return said.state(blockChangeLabel(change), blockOwnerLabel(owner));
}

export function blockTone(change: string, owner: string): Tone {
    switch (change) {
        case "removed":
            return "danger";
        case "added":
            return owner === "site" ? "info" : "ok";
        default:
            return "muted";
    }
}

export function revertedFieldLabel(outcome: string): string {
    return copy.runs.review.revert.fieldOutcomes[outcome] ?? outcome;
}

export type SentFields = { kind: "all" } | { kind: "some"; fields: readonly string[] } | { kind: "none" };

export function sentFields(offered: readonly string[], unticked: ReadonlySet<string>): SentFields {
    const keys = [...new Set(offered)];
    const ticked = keys.filter((key) => !unticked.has(key));
    if (ticked.length === keys.length) {
        return { kind: "all" };
    }
    return ticked.length === 0 ? { kind: "none" } : { kind: "some", fields: ticked };
}
