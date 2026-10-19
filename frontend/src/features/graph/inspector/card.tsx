import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../../copy/index.js";
import { errorMessageOf } from "../../../data/errors.js";
import { useUpdateEntity } from "../../../data/hooks/graph.js";
import { pushToast } from "../../../data/toasts.js";
import type { Entity } from "../../../data/types.js";
import { AddIcon, Button, CloseIcon, Field, IconButton, Input, SectionLabel, Textarea } from "../../../ui/index.js";
import type { CardDraft, FactRow, NoteRow } from "../model/card.js";
import { cardChanged, cardOf, cardProblems, cardRequest, hasProblems } from "../model/card.js";

const said = copy.graph.entityCard;

function replaced<T>(rows: readonly T[], at: number, next: T): T[] {
    return rows.map((row, index) => (index === at ? next : row));
}

function removed<T>(rows: readonly T[], at: number): T[] {
    return rows.filter((_row, index) => index !== at);
}

export interface EntityCardProps {
    entity: Entity;
}

export function EntityCard({ entity }: EntityCardProps): ReactElement {
    const update = useUpdateEntity();
    const [draft, setDraft] = useState<CardDraft>(() => cardOf(entity));
    const [tried, setTried] = useState(false);

    useEffect(() => {
        setDraft(cardOf(entity));
        setTried(false);
        update.reset();
    }, [entity.id, entity.updatedAt]);

    const problems = cardProblems(draft);
    const shown = tried ? problems : { facts: new Set<number>(), notes: new Set<number>() };
    const dirty = cardChanged(entity, draft);
    const failed = errorMessageOf(update.error);

    const editFact = (at: number, next: FactRow): void => {
        setDraft((held) => ({ ...held, facts: replaced(held.facts, at, next) }));
    };

    const editNote = (at: number, next: NoteRow): void => {
        setDraft((held) => ({ ...held, notes: replaced(held.notes, at, next) }));
    };

    const save = (): void => {
        setTried(true);
        if (hasProblems(problems)) {
            return;
        }
        const request = cardRequest(entity, draft);
        if (request === null) {
            setDraft(cardOf(entity));
            return;
        }
        update.mutate(request, {
            onSuccess: () => {
                pushToast("info", said.saved);
            },
        });
    };

    return (
        <form
            aria-label={said.title}
            className="flex flex-col gap-2.5"
            onSubmit={(event) => {
                event.preventDefault();
                save();
            }}
        >
            <p className="text-2xs text-ink-faint">{said.hint}</p>
            <Field label={said.definition} hint={said.definitionHint}>
                {(control) => (
                    <Textarea
                        {...control}
                        rows={2}
                        value={draft.definition}
                        onChange={(event) => {
                            setDraft((held) => ({ ...held, definition: event.target.value }));
                        }}
                    />
                )}
            </Field>

            <div className="flex flex-col gap-1.5">
                <SectionLabel>{said.facts}</SectionLabel>
                {draft.facts.length === 0 ? <p className="text-2xs text-ink-faint">{said.noFacts}</p> : null}
                <ul className="flex flex-col gap-1.5">
                    {draft.facts.map((fact, at) => (
                        <li key={at} className="flex flex-col gap-0.5" data-card-fact={at}>
                            <div className="flex items-center gap-1.5">
                                <div className="w-2/5 shrink-0">
                                    <Input
                                        aria-label={`${said.factLabel} ${at + 1}`}
                                        placeholder={said.factLabel}
                                        invalid={shown.facts.has(at)}
                                        value={fact.label}
                                        onChange={(event) => {
                                            editFact(at, { ...fact, label: event.target.value });
                                        }}
                                    />
                                </div>
                                <Input
                                    aria-label={`${said.factValue} ${at + 1}`}
                                    placeholder={said.factValue}
                                    invalid={shown.facts.has(at)}
                                    className="min-w-0 flex-1"
                                    value={fact.value}
                                    onChange={(event) => {
                                        editFact(at, { ...fact, value: event.target.value });
                                    }}
                                />
                                <IconButton
                                    icon={CloseIcon}
                                    label={said.removeFact(at + 1)}
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => {
                                        setDraft((held) => ({ ...held, facts: removed(held.facts, at) }));
                                    }}
                                />
                            </div>
                            {shown.facts.has(at) ? <p className="text-2xs text-danger">{said.halfBlank}</p> : null}
                        </li>
                    ))}
                </ul>
                <Button
                    type="button"
                    size="sm"
                    variant="ghost"
                    icon={AddIcon}
                    className="self-start"
                    onClick={() => {
                        setDraft((held) => ({ ...held, facts: [...held.facts, { label: "", value: "" }] }));
                    }}
                >
                    {said.addFact}
                </Button>
            </div>

            <div className="flex flex-col gap-1.5">
                <SectionLabel>{said.notes}</SectionLabel>
                {draft.notes.length === 0 ? <p className="text-2xs text-ink-faint">{said.noNotes}</p> : null}
                <ul className="flex flex-col gap-2">
                    {draft.notes.map((note, at) => (
                        <li key={at} className="flex flex-col gap-1" data-card-note={at}>
                            <div className="flex items-center gap-1.5">
                                <Input
                                    aria-label={`${said.noteLabel} ${at + 1}`}
                                    placeholder={said.noteLabel}
                                    invalid={shown.notes.has(at)}
                                    className="min-w-0 flex-1"
                                    value={note.label}
                                    onChange={(event) => {
                                        editNote(at, { ...note, label: event.target.value });
                                    }}
                                />
                                <IconButton
                                    icon={CloseIcon}
                                    label={said.removeNote(at + 1)}
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => {
                                        setDraft((held) => ({ ...held, notes: removed(held.notes, at) }));
                                    }}
                                />
                            </div>
                            <Textarea
                                aria-label={`${said.noteText} ${at + 1}`}
                                placeholder={said.noteText}
                                invalid={shown.notes.has(at)}
                                rows={2}
                                value={note.text}
                                onChange={(event) => {
                                    editNote(at, { ...note, text: event.target.value });
                                }}
                            />
                            {shown.notes.has(at) ? <p className="text-2xs text-danger">{said.halfBlank}</p> : null}
                        </li>
                    ))}
                </ul>
                <Button
                    type="button"
                    size="sm"
                    variant="ghost"
                    icon={AddIcon}
                    className="self-start"
                    onClick={() => {
                        setDraft((held) => ({ ...held, notes: [...held.notes, { label: "", text: "" }] }));
                    }}
                >
                    {said.addNote}
                </Button>
            </div>

            {failed === null ? null : <p className="text-xs text-danger">{failed}</p>}
            {dirty ? (
                <div className="flex justify-end gap-2">
                    <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                            setDraft(cardOf(entity));
                            setTried(false);
                            update.reset();
                        }}
                    >
                        {said.discard}
                    </Button>
                    <Button type="submit" variant="primary" size="sm" busy={update.isPending} data-card-save={true}>
                        {said.save}
                    </Button>
                </div>
            ) : null}
        </form>
    );
}
