import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../copy/index.js";
import { fieldErrorOf, formErrorOf } from "../../data/errors.js";
import { useSaveMapping } from "../../data/hooks/imports.js";
import type { ImportMapping, ImportOptions } from "../../data/types.js";
import { Banner, Button, Field, Input, Panel, PanelHeader } from "../../ui/index.js";
import { usable } from "./columns.js";

const separators: readonly [keyof ImportOptions, string, string][] = [
    ["keywordSeparator", copy.imports.columns.keywordSeparator, ","],
    ["anchorSeparator", copy.imports.columns.anchorSeparator, "|"],
    ["listSeparator", copy.imports.columns.listSeparator, ","],
];

export interface OptionsPanelProps {
    siteId: string;
    mapping: ImportMapping;
    options: ImportOptions;
    onOptions: (next: ImportOptions) => void;
    onSaved: (mapping: ImportMapping) => void;
}

export function OptionsPanel({ siteId, mapping, options, onOptions, onSaved }: OptionsPanelProps): ReactElement {
    const save = useSaveMapping();
    const [name, setName] = useState("");
    const thrown = save.error;

    return (
        <div className="flex flex-col gap-3 p-3">
            <Panel>
                <PanelHeader title={copy.imports.columns.options} />
                <div className="flex flex-col gap-3 p-3">
                    <Field label={copy.imports.columns.pathPrefixStrip}>
                        {(control) => (
                            <Input
                                id={control.id}
                                mono={true}
                                value={options.pathPrefixStrip ?? ""}
                                placeholder="https://example.com"
                                onChange={(event) => {
                                    onOptions({ ...options, pathPrefixStrip: event.target.value });
                                }}
                            />
                        )}
                    </Field>
                    {separators.map(([key, label, fallback]) => (
                        <Field key={key} label={label}>
                            {(control) => (
                                <Input
                                    id={control.id}
                                    mono={true}
                                    value={options[key] ?? ""}
                                    placeholder={fallback}
                                    onChange={(event) => {
                                        onOptions({ ...options, [key]: event.target.value });
                                    }}
                                />
                            )}
                        </Field>
                    ))}
                </div>
            </Panel>
            <Panel>
                <PanelHeader title={copy.imports.mappings.saveAs} />
                <div className="flex flex-col gap-2 p-3">
                    <Field label={copy.imports.mappings.saveAs} error={fieldErrorOf(thrown, "name")}>
                        {(control) => (
                            <Input
                                id={control.id}
                                data-mapping-name={true}
                                value={name}
                                invalid={control.invalid}
                                placeholder={copy.imports.mappings.namePlaceholder}
                                onChange={(event) => {
                                    setName(event.target.value);
                                }}
                            />
                        )}
                    </Field>
                    {formErrorOf(thrown) === null ? null : (
                        <Banner tone="danger" title={formErrorOf(thrown) ?? ""} />
                    )}
                    <Button
                        variant="secondary"
                        data-mapping-save={true}
                        busy={save.isPending}
                        disabled={name.trim() === "" || !usable(mapping.columns)}
                        onClick={() => {
                            save.mutate(
                                { mapping: { ...mapping, name: name.trim(), siteId } },
                                {
                                    onSuccess: (answered) => {
                                        setName("");
                                        onSaved(answered.mapping);
                                    },
                                },
                            );
                        }}
                    >
                        {copy.imports.mappings.save}
                    </Button>
                </div>
            </Panel>
        </div>
    );
}
