import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../../copy/index.js";
import { formErrorOf } from "../../../data/errors.js";
import { useUpsertModel } from "../../../data/hooks/models.js";
import type { CatalogModel } from "../../../data/types.js";
import { reasoningEfforts } from "../../../generated/vocab.js";
import { Banner, Button, Field, Input, Select, Switch } from "../../../ui/index.js";
import type { SelectOption } from "../../../ui/index.js";

const said = copy.settings.models.catalog;

const providerDefault = "default";

const effortOptions: readonly SelectOption<string>[] = [
    { value: providerDefault, label: said.effortDefault },
    ...reasoningEfforts.map((effort) => ({ value: effort, label: said.effortLabels[effort] })),
];

type Numeric = "contextTokens" | "maxOutputTokens" | "inputUsdPerM" | "outputUsdPerM" | "rpm" | "tpm";

const numericFields: readonly Numeric[] = [
    "contextTokens",
    "maxOutputTokens",
    "inputUsdPerM",
    "outputUsdPerM",
    "rpm",
    "tpm",
];

interface Draft {
    provider: string;
    model: string;
    contextTokens: string;
    maxOutputTokens: string;
    inputUsdPerM: string;
    outputUsdPerM: string;
    rpm: string;
    tpm: string;
    supportsStructured: boolean;
    supportsImages: boolean;
    reasoning: boolean;
    reasoningEffort: string;
}

function draftOf(model: CatalogModel | null): Draft {
    return {
        provider: model?.provider ?? "",
        model: model?.model ?? "",
        contextTokens: String(model?.contextTokens ?? ""),
        maxOutputTokens: String(model?.maxOutputTokens ?? ""),
        inputUsdPerM: String(model?.inputUsdPerM ?? ""),
        outputUsdPerM: String(model?.outputUsdPerM ?? ""),
        rpm: String(model?.rpm ?? ""),
        tpm: String(model?.tpm ?? ""),
        supportsStructured: model?.supportsStructured ?? false,
        supportsImages: model?.supportsImages ?? false,
        reasoning: model?.reasoning ?? false,
        reasoningEffort: model?.reasoningEffort ?? "",
    };
}

function numberOf(text: string): number | null {
    const held = Number(text.trim());
    return text.trim() === "" || !Number.isFinite(held) || held < 0 ? null : held;
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

    const missing = draft.provider.trim() === "" || draft.model.trim() === "";
    const numbers = numericFields.map((field) => numberOf(draft[field]));
    const ready = !missing && numbers.every((held) => held !== null);

    return (
        <div className="flex h-full min-h-0 flex-col">
            <div className="min-h-0 flex-1 overflow-auto p-3">
                <div className="grid grid-cols-2 gap-3">
                    <Field
                        label={said.field.provider}
                        required={true}
                        error={touched && draft.provider.trim() === "" ? copy.settings.problem.empty : null}
                    >
                        {(binding) => (
                            <Input
                                id={binding.id}
                                mono={true}
                                invalid={binding.invalid}
                                disabled={editing !== null}
                                value={draft.provider}
                                onChange={(event) => {
                                    set("provider", event.target.value);
                                }}
                            />
                        )}
                    </Field>
                    <Field
                        label={said.field.model}
                        required={true}
                        error={touched && draft.model.trim() === "" ? copy.settings.problem.empty : null}
                    >
                        {(binding) => (
                            <Input
                                id={binding.id}
                                mono={true}
                                invalid={binding.invalid}
                                disabled={editing !== null}
                                value={draft.model}
                                onChange={(event) => {
                                    set("model", event.target.value);
                                }}
                            />
                        )}
                    </Field>
                    {numericFields.map((field, index) => (
                        <Field
                            key={field}
                            label={said.field[field]}
                            required={true}
                            error={touched && numbers[index] === null ? copy.settings.problem.number : null}
                        >
                            {(binding) => (
                                <Input
                                    id={binding.id}
                                    mono={true}
                                    inputMode="decimal"
                                    invalid={binding.invalid}
                                    value={draft[field]}
                                    onChange={(event) => {
                                        set(field, event.target.value);
                                    }}
                                />
                            )}
                        </Field>
                    ))}
                </div>
                <div className="mt-3 flex flex-wrap gap-4">
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
                {draft.reasoning ? (
                    <div className="mt-3 max-w-64">
                        <Field label={said.field.effort} hint={said.effortHint}>
                            {(binding) => (
                                <Select
                                    id={binding.id}
                                    aria-describedby={binding["aria-describedby"]}
                                    value={draft.reasoningEffort === "" ? providerDefault : draft.reasoningEffort}
                                    options={effortOptions}
                                    onValueChange={(next) => {
                                        set("reasoningEffort", next === providerDefault ? "" : next);
                                    }}
                                />
                            )}
                        </Field>
                    </div>
                ) : null}
                {formErrorOf(save.error) === null ? null : (
                    <div className="pt-3">
                        <Banner tone="danger" title={formErrorOf(save.error) ?? ""} />
                    </div>
                )}
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
                        if (!ready) {
                            return;
                        }
                        save.mutate(
                            {
                                provider: draft.provider.trim(),
                                model: draft.model.trim(),
                                contextTokens: numbers[0] ?? 0,
                                maxOutputTokens: numbers[1] ?? 0,
                                inputUsdPerM: numbers[2] ?? 0,
                                outputUsdPerM: numbers[3] ?? 0,
                                rpm: numbers[4] ?? 0,
                                tpm: numbers[5] ?? 0,
                                supportsStructured: draft.supportsStructured,
                                supportsImages: draft.supportsImages,
                                reasoning: draft.reasoning,
                                reasoningEffort: draft.reasoning ? draft.reasoningEffort : "",
                            },
                            { onSuccess: onDone },
                        );
                    }}
                >
                    {copy.app.save}
                </Button>
            </div>
        </div>
    );
}
