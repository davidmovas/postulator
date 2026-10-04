import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../../copy/index.js";
import { fieldErrorOf, formErrorOf } from "../../../data/errors.js";
import { useUpsertModel } from "../../../data/hooks/models.js";
import type { CatalogModel } from "../../../data/types.js";
import { Banner, Button, Field, Input, SectionLabel, Switch } from "../../../ui/index.js";

const said = copy.settings.models.catalog;

const openaiProvider = "openai";

type Limit = "contextTokens" | "maxOutputTokens" | "rpm" | "tpm";

type Price =
    | "inputUsdPerM"
    | "cachedInputUsdPerM"
    | "cacheWriteUsdPerM"
    | "outputUsdPerM"
    | "flexInputUsdPerM"
    | "flexCachedInputUsdPerM"
    | "flexCacheWriteUsdPerM"
    | "flexOutputUsdPerM";

const limitFields: readonly Limit[] = ["contextTokens", "maxOutputTokens", "rpm", "tpm"];

const standardPrices: readonly Price[] = ["inputUsdPerM", "cachedInputUsdPerM", "cacheWriteUsdPerM", "outputUsdPerM"];

const flexPrices: readonly Price[] = [
    "flexInputUsdPerM",
    "flexCachedInputUsdPerM",
    "flexCacheWriteUsdPerM",
    "flexOutputUsdPerM",
];

const optionalFields: ReadonlySet<Limit | Price> = new Set<Limit | Price>([
    "cachedInputUsdPerM",
    "cacheWriteUsdPerM",
    ...flexPrices,
]);

type Numbers = Record<Limit | Price, string>;

interface Draft extends Numbers {
    model: string;
    supportsStructured: boolean;
    supportsImages: boolean;
    reasoning: boolean;
}

interface ModelRequest {
    provider: string;
    model: string;
    contextTokens: number;
    maxOutputTokens: number;
    inputUsdPerM: number;
    cachedInputUsdPerM: number;
    cacheWriteUsdPerM: number;
    outputUsdPerM: number;
    flexInputUsdPerM: number;
    flexCachedInputUsdPerM: number;
    flexCacheWriteUsdPerM: number;
    flexOutputUsdPerM: number;
    rpm: number;
    tpm: number;
    supportsStructured: boolean;
    supportsImages: boolean;
    reasoning: boolean;
}

function shown(model: CatalogModel | null, field: Limit | Price): string {
    const value = model?.[field];
    if (value === undefined || (optionalFields.has(field) && value === 0)) {
        return "";
    }
    return String(value);
}

function draftOf(model: CatalogModel | null): Draft {
    return {
        model: model?.model ?? "",
        contextTokens: shown(model, "contextTokens"),
        maxOutputTokens: shown(model, "maxOutputTokens"),
        rpm: shown(model, "rpm"),
        tpm: shown(model, "tpm"),
        inputUsdPerM: shown(model, "inputUsdPerM"),
        cachedInputUsdPerM: shown(model, "cachedInputUsdPerM"),
        cacheWriteUsdPerM: shown(model, "cacheWriteUsdPerM"),
        outputUsdPerM: shown(model, "outputUsdPerM"),
        flexInputUsdPerM: shown(model, "flexInputUsdPerM"),
        flexCachedInputUsdPerM: shown(model, "flexCachedInputUsdPerM"),
        flexCacheWriteUsdPerM: shown(model, "flexCacheWriteUsdPerM"),
        flexOutputUsdPerM: shown(model, "flexOutputUsdPerM"),
        supportsStructured: model?.supportsStructured ?? false,
        supportsImages: model?.supportsImages ?? false,
        reasoning: model?.reasoning ?? false,
    };
}

function numberOf(draft: Draft, field: Limit | Price): number | null {
    const trimmed = draft[field].trim();
    if (trimmed === "") {
        return optionalFields.has(field) ? 0 : null;
    }
    const held = Number(trimmed);
    return Number.isFinite(held) && held >= 0 ? held : null;
}

function requestOf(draft: Draft): ModelRequest | null {
    const held: Partial<Record<Limit | Price, number>> = {};
    for (const field of [...limitFields, ...standardPrices, ...flexPrices]) {
        const value = numberOf(draft, field);
        if (value === null) {
            return null;
        }
        held[field] = value;
    }
    const model = draft.model.trim();
    if (model === "") {
        return null;
    }
    return {
        provider: openaiProvider,
        model,
        contextTokens: held.contextTokens ?? 0,
        maxOutputTokens: held.maxOutputTokens ?? 0,
        inputUsdPerM: held.inputUsdPerM ?? 0,
        cachedInputUsdPerM: held.cachedInputUsdPerM ?? 0,
        cacheWriteUsdPerM: held.cacheWriteUsdPerM ?? 0,
        outputUsdPerM: held.outputUsdPerM ?? 0,
        flexInputUsdPerM: held.flexInputUsdPerM ?? 0,
        flexCachedInputUsdPerM: held.flexCachedInputUsdPerM ?? 0,
        flexCacheWriteUsdPerM: held.flexCacheWriteUsdPerM ?? 0,
        flexOutputUsdPerM: held.flexOutputUsdPerM ?? 0,
        rpm: held.rpm ?? 0,
        tpm: held.tpm ?? 0,
        supportsStructured: draft.supportsStructured,
        supportsImages: draft.supportsImages,
        reasoning: draft.reasoning,
    };
}

export interface ModelFormProps {
    editing: CatalogModel | null;
    onDone: () => void;
}

export function ModelForm({ editing, onDone }: ModelFormProps): ReactElement {
    const [draft, setDraft] = useState<Draft>(() => draftOf(editing));
    const [touched, setTouched] = useState(false);
    const save = useUpsertModel();

    const set = (field: keyof Draft, value: string | boolean): void => {
        setDraft({ ...draft, [field]: value });
    };

    const problemOf = (field: Limit | Price): string | null => {
        if (touched && numberOf(draft, field) === null) {
            return draft[field].trim() === "" ? copy.settings.problem.empty : copy.settings.problem.number;
        }
        return fieldErrorOf(save.error, field);
    };

    const modelProblem =
        touched && draft.model.trim() === ""
            ? copy.settings.problem.empty
            : (fieldErrorOf(save.error, "ref") ?? fieldErrorOf(save.error, "provider"));

    const numberField = (field: Limit | Price): ReactElement => (
        <Field key={field} label={said.field[field]} required={!optionalFields.has(field)} error={problemOf(field)}>
            {(binding) => (
                <Input
                    id={binding.id}
                    mono={true}
                    inputMode="decimal"
                    aria-describedby={binding["aria-describedby"]}
                    invalid={binding.invalid}
                    value={draft[field]}
                    onChange={(event) => {
                        set(field, event.target.value);
                    }}
                />
            )}
        </Field>
    );

    const formError = formErrorOf(save.error);

    return (
        <div className="flex h-full min-h-0 flex-col">
            <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-auto p-3">
                <div className="grid grid-cols-2 gap-3">
                    <Field label={said.field.model} required={true} hint={said.modelHint} error={modelProblem}>
                        {(binding) => (
                            <Input
                                id={binding.id}
                                mono={true}
                                aria-describedby={binding["aria-describedby"]}
                                invalid={binding.invalid}
                                disabled={editing !== null}
                                value={draft.model}
                                onChange={(event) => {
                                    set("model", event.target.value);
                                }}
                            />
                        )}
                    </Field>
                    <div className="flex flex-col gap-1">
                        <span className="text-xs font-medium text-ink-soft">{said.field.provider}</span>
                        <span className="flex h-7 items-center text-sm text-ink">{said.providerFixed}</span>
                    </div>
                    {limitFields.map(numberField)}
                </div>
                <section className="flex flex-col gap-2" aria-label={said.sections.standard}>
                    <SectionLabel>{said.sections.standard}</SectionLabel>
                    <div className="grid grid-cols-4 gap-3">{standardPrices.map(numberField)}</div>
                    <p className="text-xs text-ink-dim">{said.standardHelp}</p>
                </section>
                <section className="flex flex-col gap-2" aria-label={said.sections.flex}>
                    <SectionLabel>{said.sections.flex}</SectionLabel>
                    <div className="grid grid-cols-4 gap-3">{flexPrices.map(numberField)}</div>
                    <p className="text-xs text-ink-dim">{said.flexHelp}</p>
                </section>
                <div className="flex flex-wrap gap-4">
                    <Switch
                        label={said.structured}
                        checked={draft.supportsStructured}
                        onChange={(event) => {
                            set("supportsStructured", event.target.checked);
                        }}
                    />
                    <Switch
                        label={said.images}
                        checked={draft.supportsImages}
                        onChange={(event) => {
                            set("supportsImages", event.target.checked);
                        }}
                    />
                    <Switch
                        label={said.reasoning}
                        checked={draft.reasoning}
                        onChange={(event) => {
                            set("reasoning", event.target.checked);
                        }}
                    />
                </div>
                {draft.reasoning ? <p className="text-xs text-ink-dim">{said.reasoningHint}</p> : null}
                {formError === null ? null : <Banner tone="danger" title={formError} />}
            </div>
            <div className="flex shrink-0 items-center justify-end gap-2 border-t border-hairline p-3">
                <Button variant="ghost" onClick={onDone}>
                    {copy.app.cancel}
                </Button>
                <Button
                    variant="primary"
                    busy={save.isPending}
                    onClick={() => {
                        setTouched(true);
                        const request = requestOf(draft);
                        if (request === null) {
                            return;
                        }
                        save.mutate(request, { onSuccess: onDone });
                    }}
                >
                    {copy.app.save}
                </Button>
            </div>
        </div>
    );
}
