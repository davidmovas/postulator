import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../../copy/index.js";
import { fieldErrorOf, formErrorOf } from "../../../data/errors.js";
import { useSetScope } from "../../../data/hooks/templates.js";
import { pushToast } from "../../../data/toasts.js";
import type { PageType } from "../../../data/types.js";
import type { SelectOption } from "../../../ui/index.js";
import { Button, Field, Input, Select } from "../../../ui/index.js";
import { providerLabel } from "./labels.js";
import type { MetaProblem } from "./model.js";
import { customScope, metaProblem } from "./model.js";

const said = copy.pageTypes.custom;

const customProviders = ["plugin", "store"] as const;

const providerOptions: readonly SelectOption<string>[] = customProviders.map((value) => ({
    value,
    label: providerLabel(value),
}));

export interface CustomFieldFormProps {
    type: PageType;
}

export function CustomFieldForm({ type }: CustomFieldFormProps): ReactElement {
    const save = useSetScope();
    const [key, setKey] = useState("");
    const [label, setLabel] = useState("");
    const [provider, setProvider] = useState<string>(customProviders[0]);
    const [problem, setProblem] = useState<MetaProblem | null>(null);

    const submit = (): void => {
        const found = metaProblem(key, label, type.scopes ?? []);
        setProblem(found);
        if (found !== null) {
            return;
        }
        save.mutate(customScope(type.id, key, label, provider), {
            onSuccess: () => {
                pushToast("info", said.added(label.trim()));
                setKey("");
                setLabel("");
                setProblem(null);
            },
        });
    };

    const keyError =
        problem === "keyMissing" || problem === "keyShape" || problem === "taken"
            ? said[problem]
            : (fieldErrorOf(save.error, "field") ?? null);
    const labelError = problem === "labelMissing" ? said.labelMissing : fieldErrorOf(save.error, "label");
    const otherError = formErrorOf(save.error) ?? fieldErrorOf(save.error, "enabled");

    return (
        <form
            aria-label={said.title}
            className="flex flex-col gap-2.5 rounded-md border border-dashed border-edge p-3"
            onSubmit={(event) => {
                event.preventDefault();
                submit();
            }}
        >
            <div className="flex flex-col gap-0.5">
                <h3 className="text-sm font-semibold text-ink">{said.title}</h3>
                <p className="text-2xs text-ink-faint">{said.body}</p>
            </div>
            <div className="grid grid-cols-2 gap-2">
                <Field label={said.key} hint={said.keyHint} required={true} error={keyError}>
                    {(control) => (
                        <Input
                            {...control}
                            mono={true}
                            value={key}
                            data-custom-key={true}
                            onChange={(event) => {
                                setKey(event.target.value);
                            }}
                        />
                    )}
                </Field>
                <Field label={said.label} hint={said.labelHint} required={true} error={labelError}>
                    {(control) => (
                        <Input
                            {...control}
                            value={label}
                            data-custom-label={true}
                            onChange={(event) => {
                                setLabel(event.target.value);
                            }}
                        />
                    )}
                </Field>
            </div>
            <Field label={said.provider} error={fieldErrorOf(save.error, "provider")}>
                {(control) => (
                    <Select
                        id={control.id}
                        aria-describedby={control["aria-describedby"]}
                        invalid={control.invalid}
                        value={provider}
                        options={providerOptions}
                        onValueChange={setProvider}
                    />
                )}
            </Field>
            {otherError === null ? null : <p className="text-xs text-danger">{otherError}</p>}
            <div className="flex justify-end">
                <Button type="submit" size="sm" busy={save.isPending} data-custom-add={true}>
                    {said.add}
                </Button>
            </div>
        </form>
    );
}
