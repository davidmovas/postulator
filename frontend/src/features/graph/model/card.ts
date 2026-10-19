import type { Entity } from "../../../data/types.js";

export interface FactRow {
    label: string;
    value: string;
}

export interface NoteRow {
    label: string;
    text: string;
}

export interface CardDraft {
    definition: string;
    facts: FactRow[];
    notes: NoteRow[];
}

export interface CardProblems {
    facts: ReadonlySet<number>;
    notes: ReadonlySet<number>;
}

export interface CardRequest {
    id: string;
    definition: string | null;
    facts: FactRow[] | null;
    notes: NoteRow[] | null;
}

export function cardOf(entity: Entity): CardDraft {
    return {
        definition: entity.definition,
        facts: (entity.facts ?? []).map((fact) => ({ label: fact.label, value: fact.value })),
        notes: (entity.notes ?? []).map((note) => ({ label: note.label, text: note.text })),
    };
}

function blank(text: string): boolean {
    return text.trim() === "";
}

function halfBlank(left: string, right: string): boolean {
    return blank(left) !== blank(right);
}

export function cardProblems(draft: CardDraft): CardProblems {
    const facts = new Set<number>();
    const notes = new Set<number>();
    draft.facts.forEach((fact, at) => {
        if (halfBlank(fact.label, fact.value)) {
            facts.add(at);
        }
    });
    draft.notes.forEach((note, at) => {
        if (halfBlank(note.label, note.text)) {
            notes.add(at);
        }
    });
    return { facts, notes };
}

export function hasProblems(problems: CardProblems): boolean {
    return problems.facts.size > 0 || problems.notes.size > 0;
}

function keptFacts(rows: readonly FactRow[]): FactRow[] {
    return rows
        .filter((fact) => !blank(fact.label) || !blank(fact.value))
        .map((fact) => ({ label: fact.label.trim(), value: fact.value.trim() }));
}

function keptNotes(rows: readonly NoteRow[]): NoteRow[] {
    return rows
        .filter((note) => !blank(note.label) || !blank(note.text))
        .map((note) => ({ label: note.label.trim(), text: note.text.trim() }));
}

function sameFacts(left: readonly FactRow[], right: readonly FactRow[]): boolean {
    return left.length === right.length && left.every((fact, at) => fact.label === right[at]?.label && fact.value === right[at]?.value);
}

function sameNotes(left: readonly NoteRow[], right: readonly NoteRow[]): boolean {
    return left.length === right.length && left.every((note, at) => note.label === right[at]?.label && note.text === right[at]?.text);
}

export function cardChanged(entity: Entity, draft: CardDraft): boolean {
    const base = cardOf(entity);
    return (
        draft.definition !== base.definition ||
        !sameFacts(draft.facts, base.facts) ||
        !sameNotes(draft.notes, base.notes)
    );
}

export function cardRequest(entity: Entity, draft: CardDraft): CardRequest | null {
    const base = cardOf(entity);
    const definition = draft.definition.trim();
    const facts = keptFacts(draft.facts);
    const notes = keptNotes(draft.notes);
    const request: CardRequest = {
        id: entity.id,
        definition: definition === base.definition ? null : definition,
        facts: sameFacts(facts, base.facts) ? null : facts,
        notes: sameNotes(notes, base.notes) ? null : notes,
    };
    return request.definition === null && request.facts === null && request.notes === null ? null : request;
}
