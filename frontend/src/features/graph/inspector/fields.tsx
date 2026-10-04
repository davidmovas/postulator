import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../../copy/index.js";
import { fieldErrorOf, formErrorOf } from "../../../data/errors.js";
import { useUpdateEntity } from "../../../data/hooks/graph.js";
import type { Entity, Keyword } from "../../../data/types.js";
import { keywordList, sameKeywords } from "../../../domain/keywords.js";
import { entityKinds } from "../../../generated/vocab.js";
import { Button, Field, Input, KeywordInput, Select, Textarea } from "../../../ui/index.js";
import { entityKindLabel } from "../labels.js";
import type { SelectOption } from "../../../ui/index.js";

const kindOptions: readonly SelectOption<string>[] = entityKinds.map((value) => ({
    value,
    label: entityKindLabel(value),
}));

interface Draft {
    name: string;
    kind: string;
    intent: string;
    keywords: Keyword[];
}

function draftOf(entity: Entity): Draft {
    return {
        name: entity.name,
        kind: entity.kind,
        intent: entity.intent,
        keywords: keywordList(entity.keywords),
    };
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
        !sameKeywords(draft.keywords, base.keywords);

    const save = (): void => {
        update.mutate({
            id: entity.id,
            name: draft.name !== base.name ? draft.name : null,
            kind: draft.kind !== base.kind ? draft.kind : null,
            intent: draft.intent !== base.intent ? draft.intent : null,
            keywords: sameKeywords(draft.keywords, base.keywords) ? null : draft.keywords,
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
            <Field label={copy.graph.form.keywords} hint={copy.graph.form.keywordsHint} error={fieldErrorOf(update.error, "keywords")}>
                {(control) => (
                    <KeywordInput
                        id={control.id}
                        aria-describedby={control["aria-describedby"]}
                        invalid={control.invalid}
                        values={draft.keywords}
                        removeLabel={copy.graph.form.remove}
                        phraseLabel={copy.graph.form.keyword}
                        volumeLabel={copy.graph.form.volume}
                        onChange={(keywords) => {
                            edit({ keywords });
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
