import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { useMappings } from "../../data/hooks/imports.js";
import type { ImportMapping, ImportOptions, ImportSheet } from "../../data/types.js";
import type { ImportRowType } from "../../generated/vocab.js";
import { importRowTypes, isOneOf } from "../../generated/vocab.js";
import { Field, Input, Panel, PanelHeader, SectionLabel, Segmented, Select, Switch } from "../../ui/index.js";
import { freeHeaders, groupHeaders, toggled } from "./columns.js";
import { rowTypeLabel } from "./labels.js";
import type { SheetSettings } from "./workbook.js";

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
    empty: string;
    onToggle: (header: string) => void;
}

function ColumnChecklist({ title, hint, choices, chosen, empty, onToggle }: ColumnChecklistProps): ReactElement {
    return (
        <Panel>
            <PanelHeader title={title} />
            <div className="flex flex-col gap-2 p-3">
                <p className="text-2xs text-ink-faint">{hint}</p>
                {choices.length === 0 ? (
                    <p className="text-2xs text-ink-dim">{empty}</p>
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

interface SheetSwitchesProps {
    sheets: readonly ImportSheet[];
    chosen: readonly string[];
    onChoose: (sheet: string, on: boolean) => void;
}

function SheetSwitches({ sheets, chosen, onChoose }: SheetSwitchesProps): ReactElement {
    return (
        <Panel>
            <PanelHeader title={copy.imports.columns.sheets} />
            <div className="flex flex-col gap-1 p-3">
                <p className="text-2xs text-ink-faint">{copy.imports.columns.sheetsHint}</p>
                <p className="text-2xs text-ink-dim">{copy.imports.columns.reading(chosen)}</p>
                <ul className="flex flex-col">
                    {sheets.map((sheet) => (
                        <li key={sheet.name} className="flex h-7 items-center gap-2">
                            <Switch
                                label={sheet.name}
                                checked={chosen.includes(sheet.name)}
                                onChange={(event) => {
                                    onChoose(sheet.name, event.target.checked);
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
    );
}

interface SavedPickerProps {
    siteId: string;
    mappingId: string;
    onPick: (saved: ImportMapping) => void;
}

function SavedPicker({ siteId, mappingId, onPick }: SavedPickerProps): ReactElement | null {
    const mappings = useMappings(siteId === "" ? null : siteId);
    const saved = (mappings.data?.mappings ?? []).filter((mapping) => (mapping.id ?? "") !== "");
    if (saved.length === 0) {
        return null;
    }
    return (
        <Panel>
            <PanelHeader title={copy.imports.columns.saved} />
            <div className="flex flex-col gap-2 p-3">
                <Select<string>
                    aria-label={copy.imports.columns.saved}
                    size="sm"
                    value={mappingId === "" ? null : mappingId}
                    placeholder={copy.imports.columns.savedPlaceholder}
                    options={saved.map((mapping) => ({ value: mapping.id ?? "", label: mapping.name ?? "" }))}
                    onValueChange={(picked) => {
                        const found = saved.find((mapping) => mapping.id === picked);
                        if (found !== undefined) {
                            onPick(found);
                        }
                    }}
                />
                <p className="text-2xs text-ink-faint">{copy.imports.columns.savedHint}</p>
            </div>
        </Panel>
    );
}

export interface OptionsPanelProps {
    siteId: string;
    sheets: readonly ImportSheet[];
    chosen: readonly string[];
    active: string;
    settings: SheetSettings;
    headers: readonly string[];
    onChoose: (sheet: string, on: boolean) => void;
    onOptions: (next: ImportOptions) => void;
    onHeaderless: (on: boolean) => void;
    onSaved: (saved: ImportMapping) => void;
}

export function OptionsPanel({
    siteId,
    sheets,
    chosen,
    active,
    settings,
    headers,
    onChoose,
    onOptions,
    onHeaderless,
    onSaved,
}: OptionsPanelProps): ReactElement {
    const { columns, options } = settings;
    const workbook = sheets.length > 1;
    const open = chosen.includes(active);
    const indent = options.indentColumns ?? [];
    const levels = options.levelColumns ?? [];
    const notes = options.noteColumns ?? [];
    const roots = sheets.find((sheet) => sheet.name === active)?.detected.options.levelColumns ?? [];

    return (
        <div className="flex flex-col gap-3 p-3">
            {workbook ? <SheetSwitches sheets={sheets} chosen={chosen} onChoose={onChoose} /> : null}
            {!open ? null : (
                <>
                    {workbook ? (
                        <SectionLabel className="px-1 pt-1">{copy.imports.columns.settingsOf(active)}</SectionLabel>
                    ) : null}
                    <SavedPicker siteId={siteId} mappingId={settings.mappingId} onPick={onSaved} />
                    <Panel>
                        <PanelHeader title={copy.imports.columns.indent} />
                        <div className="flex flex-col gap-2 p-3">
                            <Switch
                                data-no-header={true}
                                label={copy.imports.columns.noHeader}
                                title={copy.imports.columns.noHeaderHint}
                                checked={options.noHeader === true}
                                onChange={(event) => {
                                    onHeaderless(event.target.checked);
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
                                                onOptions({ ...options, indentColumns: toggled(indent, header, headers) });
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
                        choices={groupHeaders(headers, columns, notes, roots)}
                        chosen={levels}
                        empty={copy.imports.columns.noGroupColumns}
                        onToggle={(header) => {
                            onOptions({ ...options, levelColumns: toggled(levels, header, headers) });
                        }}
                    />
                    <ColumnChecklist
                        title={copy.imports.columns.notes}
                        hint={copy.imports.columns.notesHint}
                        choices={freeHeaders(headers, columns, levels)}
                        chosen={notes}
                        empty={copy.imports.columns.noFreeColumns}
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
                </>
            )}
        </div>
    );
}
