import { useSyncExternalStore } from "react";

import type { PreviewFromPagesRequest, ProposedEntity } from "../../../data/types.js";
import type { ProposalEdit, ProposalEdits, ProposalFilter } from "../model/proposal-tree.js";
import type { Pick, ProposalInput } from "./propose-pick.js";
import { freshPick } from "./propose-pick.js";

export type ProposalOpener = ProposalInput | "seed";

export type ProposalStage = "pick" | "review" | "confirm";

export interface ProposalMemory {
    stage: ProposalStage;
    pick: Pick;
    proposals: readonly ProposedEntity[];
    kept: ReadonlySet<number>;
    edits: ProposalEdits;
    query: string;
    filter: ProposalFilter;
    collapsed: ReadonlySet<number>;
    focus: number | null;
    anchor: number | null;
    pageTypes: ReadonlyMap<string, string>;
    typeChoice: string | null;
    seed: string | null;
}

function blank(input: ProposalInput, stage: ProposalStage): ProposalMemory {
    return Object.freeze({
        stage,
        pick: freshPick(input),
        proposals: [],
        kept: new Set<number>(),
        edits: new Map<number, ProposalEdit>(),
        query: "",
        filter: "all" as const,
        collapsed: new Set<number>(),
        focus: null,
        anchor: null,
        pageTypes: new Map<string, string>(),
        typeChoice: null,
        seed: null,
    });
}

const blanks: Readonly<Record<ProposalOpener, ProposalMemory>> = {
    pages: blank("pages", "pick"),
    keywords: blank("keywords", "pick"),
    seed: blank("pages", "review"),
};

export function blankMemory(opener: ProposalOpener): ProposalMemory {
    return blanks[opener];
}

export function seedKey(seed: PreviewFromPagesRequest): string {
    return JSON.stringify([seed.siteId, seed.source ?? "", [...(seed.pageIds ?? [])].sort()]);
}

const memories = new Map<string, ProposalMemory>();
const listeners = new Set<() => void>();

function keyOf(siteId: string, opener: ProposalOpener): string {
    return `${siteId}\u0000${opener}`;
}

function announce(): void {
    for (const listener of listeners) {
        listener();
    }
}

function subscribe(listener: () => void): () => void {
    listeners.add(listener);
    return () => {
        listeners.delete(listener);
    };
}

export function memoryOf(siteId: string, opener: ProposalOpener): ProposalMemory {
    return memories.get(keyOf(siteId, opener)) ?? blanks[opener];
}

export function remember(siteId: string, opener: ProposalOpener, patch: Partial<ProposalMemory>): void {
    memories.set(keyOf(siteId, opener), Object.freeze({ ...memoryOf(siteId, opener), ...patch }));
    announce();
}

export function forgetProposal(siteId: string, opener: ProposalOpener): void {
    if (memories.delete(keyOf(siteId, opener))) {
        announce();
    }
}

export function useProposalMemory(siteId: string, opener: ProposalOpener): ProposalMemory {
    return useSyncExternalStore(subscribe, () => memoryOf(siteId, opener));
}
