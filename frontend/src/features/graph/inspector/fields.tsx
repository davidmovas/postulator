import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../../copy/index.js";
import { react } from "../../../data/errors.js";
import { useUpdateEntity } from "../../../data/hooks/graph.js";
import type { Entity } from "../../../data/types.js";
import { entityKinds } from "../../../generated/vocab.js";
import { Button, ChipInput, Field, Input, Select, Textarea } from "../../../ui/index.js";
import { kindLabel } from "../labels.js";
import type { SelectOption } from "../../../ui/index.js";

const kindOptions: readonly SelectOption<string>[] = entityKinds.map((value) => ({
    value,
    label: kindLabel(value),
}));

interface Draft {
    name: string;
    kind: string;
    intent: string;
    primaryKeyword: string;
    secondaryKeywords: string[];
}

function draftOf(entity: Entity): Draft {
    return {
        name: entity.name,
        kind: entity.kind,
        intent: entity.intent,
        primaryKeyword: entity.primaryKeyword,
        secondaryKeywords: [...(entity.secondaryKeywords ?? [])],
    };
}

function sameList(left: readonly string[], right: readonly string[]): boolean {
    return left.length === right.length && left.every((value, index) => value === right[index]);
}

export function fieldErrorOf(thrown: unknown, field: string): string | null {
    if (thrown === null || thrown === undefined) {
        return null;
    }
    const reaction = react(thrown);
    return reaction.kind === "field" && reaction.field === field ? reaction.message : null;
}

export function formErrorOf(thrown: unknown): string | null {
    if (thrown === null || thrown === undefined) {
        return null;
    }
    const reaction = react(thrown);
    return reaction.kind === "silent" || reaction.kind === "unlock" || reaction.kind === "field" ? null : reaction.message;
}

export interface EntityFieldsProps {
    entity: Entity;
}

export function EntityFields({ entity }: EntityFieldsProps): ReactElement {
    const update = useUpdateEntity();
    const [draft, setDraft] = useState<Draft>(() => draftOf(entity));

    useEffect(() => {
        setDraft(draftOf(entity));
        update.reset();
    }, [entity.id, entity.updatedAt]);

    const edit = (patch: Partial<Draft>): void => {
        setDraft((held) => ({ ...held, ...patch }));
    };

    const base = draftOf(entity);
    const dirty =
        draft.name !== base.name ||
        draft.kind !== base.kind ||
        draft.intent !== base.intent ||
        draft.primaryKeyword !== base.primaryKeyword ||
        !sameList(draft.secondaryKeywords, base.secondaryKeywords);

    const save = (): void => {
        update.mutate({
            id: entity.id,
            name: draft.name !== base.name ? draft.name : null,
            kind: draft.kind !== base.kind ? draft.kind : null,
            intent: draft.intent !== base.intent ? draft.intent : null,
            primaryKeyword: draft.primaryKeyword !== base.primaryKeyword ? draft.primaryKeyword : null,
            secondaryKeywords: sameList(draft.secondaryKeywords, base.secondaryKeywords) ? null : draft.secondaryKeywords,
        });
    };

    const formError = formErrorOf(update.error);

    return (
        <form
            className="flex flex-col gap-2.5"
            onSubmit={(event) => {
                event.preventDefault();
                if (dirty) {
                    save();
                }
            }}
        >
            <Field label={copy.graph.form.name} required={true} error={fieldErrorOf(update.error, "name")}>
                {(control) => (
                    <Input
                        {...control}
                        value={draft.name}
                        onChange={(event) => {
                            edit({ name: event.target.value });
                        }}
                    />
                )}
            </Field>
            <Field label={copy.graph.form.kind} error={fieldErrorOf(update.error, "kind")}>
                {(control) => (
                    <Select
                        id={control.id}
                        aria-describedby={control["aria-describedby"]}
                        invalid={control.invalid}
                        value={draft.kind}
                        options={kindOptions}
                        onValueChange={(kind) => {
                            edit({ kind });
                        }}
                    />
                )}
            </Field>
            <Field label={copy.graph.form.primaryKeyword} error={fieldErrorOf(update.error, "primaryKeyword")}>
                {(control) => (
                    <Input
                        {...control}
                        mono={true}
                        value={draft.primaryKeyword}
                        onChange={(event) => {
                            edit({ primaryKeyword: event.target.value });
                        }}
                    />
                )}
            </Field>
            <Field
                label={copy.graph.form.secondaryKeywords}
                hint={copy.graph.form.keywordsHint}
                error={fieldErrorOf(update.error, "secondaryKeywords")}
            >
                {(control) => (
                    <ChipInput
                        id={control.id}
                        aria-describedby={control["aria-describedby"]}
                        invalid={control.invalid}
                        mono={true}
                        values={draft.secondaryKeywords}
                        removeLabel={copy.graph.form.remove}
                        onChange={(secondaryKeywords) => {
                            edit({ secondaryKeywords });
                        }}
                    />
                )}
            </Field>
            <Field label={copy.graph.form.intent} hint={copy.graph.form.intentHint} error={fieldErrorOf(update.error, "intent")}>
                {(control) => (
                    <Textarea
                        {...control}
                        rows={2}
                        value={draft.intent}
                        onChange={(event) => {
                            edit({ intent: event.target.value });
                        }}
                    />
                )}
            </Field>
            {formError === null ? null : <p className="text-xs text-danger">{formError}</p>}
            {dirty ? (
                <div className="flex justify-end gap-2">
                    <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                            setDraft(draftOf(entity));
                            update.reset();
                        }}
                    >
                        {copy.graph.form.discard}
                    </Button>
                    <Button type="submit" variant="primary" size="sm" busy={update.isPending}>
                        {copy.graph.form.save}
                    </Button>
                </div>
            ) : null}
        </form>
    );
}
