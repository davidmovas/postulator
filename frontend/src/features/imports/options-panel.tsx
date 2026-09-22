import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../copy/index.js";
import { fieldErrorOf, formErrorOf } from "../../data/errors.js";
import { useSaveMapping } from "../../data/hooks/imports.js";
import type { ImportMapping, ImportOptions, ImportSheet } from "../../data/types.js";
import { Banner, Button, Field, Input, Panel, PanelHeader, Switch } from "../../ui/index.js";
import { usable } from "./columns.js";

function letterOf(at: number): string {
    let name = "";
    let index = at;
    while (index >= 0) {
        name = String.fromCharCode(65 + (index % 26)) + name;
        index = Math.floor(index / 26) - 1;
    }
    return name;
}

type SeparatorKey = "keywordSeparator" | "anchorSeparator" | "listSeparator";

const separators: readonly [SeparatorKey, string, string][] = [
    ["keywordSeparator", copy.imports.columns.keywordSeparator, ","],
    ["anchorSeparator", copy.imports.columns.anchorSeparator, "|"],
    ["listSeparator", copy.imports.columns.listSeparator, ","],
];

export interface OptionsPanelProps {
    siteId: string;
    mapping: ImportMapping;
    options: ImportOptions;
    sheets: readonly ImportSheet[];
    headers: readonly string[];
    onOptions: (next: ImportOptions) => void;
    onSaved: (mapping: ImportMapping) => void;
}

export function OptionsPanel({
    siteId,
    mapping,
    options,
    sheets,
    headers,
    onOptions,
    onSaved,
}: OptionsPanelProps): ReactElement {
    const save = useSaveMapping();
    const [name, setName] = useState("");
    const thrown = save.error;

    const chosenSheets = options.sheets ?? [];
    const toggleSheet = (sheet: string): void => {
        const held = new Set(chosenSheets);
        if (held.has(sheet)) {
            held.delete(sheet);
        } else {
            held.add(sheet);
        }
        onOptions({ ...options, sheets: sheets.map((each) => each.name).filter((each) => held.has(each)) });
    };

    const indent = options.indentColumns ?? [];
    const toggleIndent = (column: string): void => {
        const held = new Set(indent);
        if (held.has(column)) {
            held.delete(column);
        } else {
            held.add(column);
        }
        onOptions({ ...options, indentColumns: headers.filter((each) => held.has(each)) });
    };

    return (
        <div className="flex flex-col gap-3 p-3">
            {sheets.length < 2 ? null : (
                <Panel>
                    <PanelHeader title={copy.imports.columns.sheets} />
                    <div className="flex flex-col gap-1 p-3">
                        <p className="text-2xs text-ink-faint">{copy.imports.columns.sheetsHint}</p>
                        <ul className="flex flex-col">
                            {sheets.map((sheet) => (
                                <li key={sheet.name} className="flex h-7 items-center gap-2">
                                    <Switch
                                        label={sheet.name}
                                        checked={chosenSheets.includes(sheet.name)}
                                        onChange={() => {
                                            toggleSheet(sheet.name);
                                        }}
                                    />
                                    <span className="ml-auto shrink-0 font-mono text-2xs text-ink-faint">
                                        {copy.imports.columns.sheetRows(sheet.rows)}
                                    </span>
                                </li>
                            ))}
                        </ul>
                    </div>
                </Panel>
            )}
            <Panel>
                <PanelHeader title={copy.imports.columns.indent} />
                <div className="flex flex-col gap-2 p-3">
                    <Switch
                        data-no-header={true}
                        label={copy.imports.columns.noHeader}
                        title={copy.imports.columns.noHeaderHint}
                        checked={options.noHeader === true}
                        onChange={(event) => {
                            onOptions({ ...options, noHeader: event.target.checked, indentColumns: [] });
                        }}
                    />
                    <p className="text-2xs text-ink-faint">{copy.imports.columns.indentHint}</p>
                    <ul className="flex flex-col">
                        {headers.map((header, at) => (
                            <li key={`${header}-${String(at)}`} className="flex h-7 items-center">
                                <Switch
                                    className="w-full"
                                    label={header === "" ? copy.imports.columns.unnamed(letterOf(at)) : header}
                                    checked={indent.includes(header)}
                                    onChange={() => {
                                        toggleIndent(header);
                                    }}
                                />
                            </li>
                        ))}
                    </ul>
                </div>
            </Panel>
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
                        disabled={name.trim() === "" || !usable(mapping.columns, indent)}
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
