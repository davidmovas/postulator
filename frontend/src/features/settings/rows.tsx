import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../copy/index.js";
import { keyErrorOf } from "../../data/errors.js";
import { useSetSetting } from "../../data/hooks/settings.js";
import type { SettingDescriptor } from "../../data/types.js";
import { IconButton, Input, RestartAltIcon, Select, Switch } from "../../ui/index.js";
import { humanLabel, unitOf } from "./model/layout.js";
import { boundNumber, boundText, checkSetting, kindOf, parseValue, renderValue, sameValue } from "./model/value.js";
import type { SettingKind, SettingProblem } from "./model/value.js";
import type { SettingEntry } from "./schema.js";

type EnumLabels = Readonly<Record<string, Readonly<Record<string, string>>>>;

const enumLabels = copy.settings.enumLabels as EnumLabels;
const described = copy.settings.keys as Readonly<Record<string, { label: string; help: string }>>;

function problemText(kind: SettingKind, problem: SettingProblem): string {
    switch (problem.kind) {
        case "empty":
            return copy.settings.problem.empty;
        case "shape":
            return kind === "duration" ? copy.settings.problem.duration : copy.settings.problem.whole;
        default:
            if (problem.min !== null && problem.max !== null) {
                return copy.settings.problem.range(problem.min, problem.max);
            }
            if (problem.min !== null) {
                return copy.settings.problem.lowerBound(problem.min);
            }
            return copy.settings.problem.upperBound(problem.max ?? "");
    }
}

function bounds(descriptor: SettingDescriptor): string | null {
    const kind = kindOf(descriptor);
    const min = kind === "int" ? boundNumber(descriptor.min) : boundText(descriptor.min);
    const max = kind === "int" ? boundNumber(descriptor.max) : boundText(descriptor.max);
    if (min !== null && max !== null) {
        return copy.settings.hint.range(String(min), String(max));
    }
    if (min !== null) {
        return copy.settings.hint.lowerBound(String(min));
    }
    if (max !== null) {
        return copy.settings.hint.upperBound(String(max));
    }
    return descriptor.nonEmpty === true ? copy.settings.hint.required : null;
}

function controlWidth(kind: SettingKind): string {
    if (kind === "enum") {
        return "w-44";
    }
    return kind === "string" ? "w-72" : "w-20";
}

export function SettingRow({ entry }: { entry: SettingEntry }): ReactElement {
    const { descriptor, value, loading } = entry;
    const kind = kindOf(descriptor);
    const stored = renderValue(kind, value);
    const [draft, setDraft] = useState<string | null>(null);
    const [problem, setProblem] = useState<SettingProblem | null>(null);
    const [focused, setFocused] = useState(false);
    const write = useSetSetting();

    useEffect(() => {
        setDraft(null);
        setProblem(null);
    }, [stored]);

    const prose = described[descriptor.key];
    const label = prose === undefined ? humanLabel(descriptor.key) : prose.label;
    const unit = unitOf(descriptor.key);
    const isDefault = sameValue(kind, value, descriptor.default);
    const shown = draft ?? stored;

    const commit = (text: string): void => {
        const found = checkSetting(descriptor, text);
        setProblem(found);
        if (found !== null) {
            return;
        }
        const next = parseValue(kind, text);
        if (sameValue(kind, next, value)) {
            setDraft(null);
            return;
        }
        write.mutate({ key: descriptor.key, value: next });
    };

    const refused = problem === null ? keyErrorOf(write.error, descriptor.key) : problemText(kind, problem);
    const note = refused ?? (focused ? bounds(descriptor) : null);

    const control =
        kind === "bool" ? (
            <Switch
                label={label}
                checked={value === true}
                disabled={loading || write.isPending}
                onChange={(event) => {
                    write.mutate({ key: descriptor.key, value: event.target.checked });
                }}
            />
        ) : kind === "enum" ? (
            <Select
                aria-label={label}
                value={typeof value === "string" ? value : null}
                options={(descriptor.enum ?? []).map((member) => ({
                    value: member,
                    label: enumLabels[descriptor.key]?.[member] ?? member,
                }))}
                onValueChange={(picked) => {
                    write.mutate({ key: descriptor.key, value: picked });
                }}
            />
        ) : (
            <Input
                aria-label={label}
                invalid={refused !== null}
                mono={true}
                inputMode={kind === "int" ? "numeric" : "text"}
                value={shown}
                disabled={loading}
                onChange={(event) => {
                    setDraft(event.target.value);
                    setProblem(null);
                }}
                onFocus={() => {
                    setFocused(true);
                }}
                onBlur={(event) => {
                    setFocused(false);
                    commit(event.target.value);
                }}
                onKeyDown={(event) => {
                    if (event.key === "Enter") {
                        commit(event.currentTarget.value);
                    }
                    if (event.key === "Escape") {
                        setDraft(null);
                        setProblem(null);
                    }
                }}
            />
        );

    return (
        <div className="border-b border-hairline px-3 py-2 last:border-b-0">
            <div className="flex items-center gap-3">
                <span title={prose?.help} className="min-w-0 flex-1 truncate text-sm text-ink-soft">
                    {label}
                </span>
                <div className={controlWidth(kind)}>{control}</div>
                <span className="w-16 shrink-0 text-xs text-ink-dim">{unit === null ? "" : copy.settings.units[unit]}</span>
                <div className="flex w-24 shrink-0 items-center justify-end gap-1">
                    {isDefault ? (
                        <span className="text-2xs text-ink-faint">{copy.settings.isDefault}</span>
                    ) : (
                        <>
                            <span className="text-2xs text-accent">{copy.settings.changed}</span>
                            <IconButton
                                icon={RestartAltIcon}
                                label={copy.settings.reset}
                                variant="ghost"
                                size="sm"
                                disabled={write.isPending}
                                onClick={() => {
                                    write.mutate({ key: descriptor.key, value: descriptor.default });
                                }}
                            />
                        </>
                    )}
                </div>
            </div>
            {note === null ? null : (
                <div className="flex justify-end pt-1">
                    <p className={refused === null ? "text-xs text-ink-faint" : "text-xs text-danger"}>{note}</p>
                </div>
            )}
        </div>
    );
}

export function SettingRows({ entries }: { entries: readonly SettingEntry[] }): ReactElement {
    return (
        <>
            {entries.map((entry) => (
                <SettingRow key={entry.descriptor.key} entry={entry} />
            ))}
        </>
    );
}
