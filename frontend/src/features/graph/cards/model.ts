import type { CardFact, CardProposal, ProposedCard } from "../../../data/types.js";

export type FactMark = "same" | "changed" | "added" | "removed";

export interface MarkedFact {
    label: string;
    value: string;
    mark: FactMark;
}

function factKey(label: string): string {
    return label.trim().toLowerCase();
}

function valuesByLabel(facts: readonly CardFact[]): Map<string, string> {
    const held = new Map<string, string>();
    for (const fact of facts) {
        const key = factKey(fact.label);
        if (!held.has(key)) {
            held.set(key, fact.value.trim());
        }
    }
    return held;
}

function factsOf(card: ProposedCard): readonly CardFact[] {
    return card.facts ?? [];
}

export function currentFacts(proposal: CardProposal): MarkedFact[] {
    const proposed = valuesByLabel(factsOf(proposal.proposed));
    return factsOf(proposal.current).map((fact) => {
        const other = proposed.get(factKey(fact.label));
        const mark: FactMark = other === undefined ? "removed" : other === fact.value.trim() ? "same" : "changed";
        return { label: fact.label, value: fact.value, mark };
    });
}

export function proposedFacts(proposal: CardProposal): MarkedFact[] {
    const current = valuesByLabel(factsOf(proposal.current));
    return factsOf(proposal.proposed).map((fact) => {
        const other = current.get(factKey(fact.label));
        const mark: FactMark = other === undefined ? "added" : other === fact.value.trim() ? "same" : "changed";
        return { label: fact.label, value: fact.value, mark };
    });
}

export function definitionMoves(proposal: CardProposal): boolean {
    return proposal.current.definition.trim() !== proposal.proposed.definition.trim();
}

export function waitingEntities(proposals: readonly CardProposal[]): ReadonlySet<string> {
    return new Set(proposals.map((proposal) => proposal.entityId));
}

export type Refusal = "changed" | "replaced" | "settled" | "other";

export function refusalOf(code: string, reason: unknown): Refusal {
    if (code !== "CONFLICT") {
        return "other";
    }
    if (reason === "card_changed") {
        return "changed";
    }
    return reason === "proposal_changed" ? "replaced" : "settled";
}
