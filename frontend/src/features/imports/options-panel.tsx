import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../copy/index.js";
import { fieldErrorOf, formErrorOf } from "../../data/errors.js";
import { useSaveMapping } from "../../data/hooks/imports.js";
import type { ImportMapping, ImportOptions, ImportSheet } from "../../data/types.js";
import type { ImportRowType } from "../../generated/vocab.js";
import { importRowTypes, isOneOf } from "../../generated/vocab.js";
import { Banner, Button, Field, Input, Panel, PanelHeader, Segmented, Switch } from "../../ui/index.js";
import { freeHeaders, toggled, usable } from "./columns.js";
import { rowTypeLabel } from "./labels.js";

function letterOf(at: number): string {
    let name = "";
    let index = at;
    while (index >= 0) {
        name = String.fromCharCode(65 + (index % 26)) + name;
        index = Math.floor(index / 26) - 1;
    }
    return name;
}

function rowTypeOf(options: ImportOptions): ImportRowType {
    const held: string = options.rowType ?? "";
    return isOneOf(importRowTypes, held) ? held : "pages";
}

function wireRowType(value: ImportRowType): ImportOptions["rowType"] {
    return value as ImportOptions["rowType"];
}

type SeparatorKey = "anchorSeparator" | "listSeparator";

const separators: readonly [SeparatorKey, string, string][] = [
    ["anchorSeparator", copy.imports.columns.anchorSeparator, "|"],
    ["listSeparator", copy.imports.columns.listSeparator, ","],
];

interface ColumnChecklistProps {
    title: string;
    hint: string;
    choices: readonly string[];
    chosen: readonly string[];
    onToggle: (header: string) => void;
}

function ColumnChecklist({ title, hint, choices, chosen, onToggle }: ColumnChecklistProps): ReactElement {
    return (
        <Panel>
            <PanelHeader title={title} />
            <div className="flex flex-col gap-2 p-3">
                <p className="text-2xs text-ink-faint">{hint}</p>
                {choices.length === 0 ? (
                    <p className="text-2xs text-ink-dim">{copy.imports.columns.noFreeColumns}</p>
                ) : (
                    <ul className="flex flex-col">
                        {choices.map((header) => (
                            <li key={header} className="flex h-7 items-center">
                                <Switch
                                    className="w-full"
                                    label={header}
                                    checked={chosen.includes(header)}
                                    onChange={() => {
                                        onToggle(header);
                                    }}
                                />
                            </li>
                        ))}
                    </ul>
                )}
            </div>
        </Panel>
    );
}

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
        onOptions({
            ...options,
            sheets: sheets.map((each) => each.name).filter((each) => held.has(each)),
            indentColumns: [],
        });
    };

    const reading = chosenSheets.length > 0 ? chosenSheets : sheets.slice(0, 1).map((each) => each.name);
    const indent = options.indentColumns ?? [];
    const toggleIndent = (column: string): void => {
        onOptions({ ...options, indentColumns: toggled(indent, column, headers) });
    };
    const levels = options.levelColumns ?? [];
    const notes = options.noteColumns ?? [];

    return (
        <div className="flex flex-col gap-3 p-3">
            {sheets.length < 2 ? null : (
                <Panel>
                    <PanelHeader title={copy.imports.columns.sheets} />
                    <div className="flex flex-col gap-1 p-3">
                        <p className="text-2xs text-ink-faint">{copy.imports.columns.sheetsHint}</p>
                        <p className="text-2xs text-ink-dim">{copy.imports.columns.reading(reading)}</p>
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
                            onOptions({
                                ...options,
                                noHeader: event.target.checked,
                                indentColumns: [],
                                levelColumns: [],
                                noteColumns: [],
                            });
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
                <PanelHeader title={copy.imports.rowType.title} />
                <div className="flex flex-col gap-2 p-3">
                    <Segmented
                        label={copy.imports.rowType.title}
                        value={rowTypeOf(options)}
                        options={importRowTypes.map((value) => ({ value, label: rowTypeLabel(value) }))}
                        onValueChange={(value) => {
                            onOptions({ ...options, rowType: wireRowType(value) });
                        }}
                    />
                    <p className="text-2xs text-ink-faint">{copy.imports.rowType.hint}</p>
                    {rowTypeOf(options) === "pages" ? null : (
                        <p className="text-2xs text-ink-faint">{copy.imports.rowType.products}</p>
                    )}
                </div>
            </Panel>
            <ColumnChecklist
                title={copy.imports.columns.levels}
                hint={copy.imports.columns.levelsHint}
                choices={freeHeaders(headers, mapping.columns, notes)}
                chosen={levels}
                onToggle={(header) => {
                    onOptions({ ...options, levelColumns: toggled(levels, header, headers) });
                }}
            />
            <ColumnChecklist
                title={copy.imports.columns.notes}
                hint={copy.imports.columns.notesHint}
                choices={freeHeaders(headers, mapping.columns, levels)}
                chosen={notes}
                onToggle={(header) => {
                    onOptions({ ...options, noteColumns: toggled(notes, header, headers) });
                }}
            />
            <Panel>
                <PanelHeader title={copy.imports.columns.options} />
                <div className="flex flex-col gap-3 p-3">
                    <p className="text-2xs text-ink-faint">{copy.imports.columns.keywordsFormat}</p>
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
                        disabled={name.trim() === "" || !usable(mapping.columns, indent, levels)}
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
