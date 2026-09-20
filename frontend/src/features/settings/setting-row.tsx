import type { ReactElement } from "react";
import { useEffect, useState } from "react";

import { copy } from "../../copy/index.js";
import { keyErrorOf } from "../../data/errors.js";
import { useSetSetting } from "../../data/hooks/settings.js";
import type { SettingDescriptor } from "../../data/types.js";
import {
    Banner,
    Field,
    IconButton,
    Input,
    RestartAltIcon,
    Select,
    StatusBadge,
    Switch,
} from "../../ui/index.js";
import { boundNumber, boundText, checkSetting, kindOf, parseValue, renderValue, sameValue } from "./model/value.js";
import type { SettingKind, SettingProblem } from "./model/value.js";

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

function boundHint(descriptor: SettingDescriptor): string | undefined {
    const kind = kindOf(descriptor);
    if (kind === "int") {
        const min = boundNumber(descriptor.min);
        const max = boundNumber(descriptor.max);
        if (min !== null && max !== null) {
            return copy.settings.hint.range(String(min), String(max));
        }
        return undefined;
    }
    if (kind === "duration") {
        const min = boundText(descriptor.min);
        const max = boundText(descriptor.max);
        if (min !== null && max !== null) {
            return copy.settings.hint.range(min, max);
        }
    }
    return undefined;
}

export interface SettingRowProps {
    descriptor: SettingDescriptor;
    value: unknown;
    loading: boolean;
}

export function SettingRow({ descriptor, value, loading }: SettingRowProps): ReactElement {
    const kind = kindOf(descriptor);
    const stored = renderValue(kind, value);
    const [draft, setDraft] = useState<string | null>(null);
    const [problem, setProblem] = useState<SettingProblem | null>(null);
    const write = useSetSetting();

    useEffect(() => {
        setDraft(null);
        setProblem(null);
    }, [stored]);

    const shown = draft ?? stored;
    const prose = described[descriptor.key] ?? null;
    const isDefault = sameValue(kind, value, descriptor.default);

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
    const hint = refused === null ? (boundHint(descriptor) ?? (isDefault ? copy.settings.hint.isDefault : undefined)) : undefined;

    const control = (
        <Field
            label={prose === null ? descriptor.key : prose.label}
            hint={hint}
            error={refused}
            required={descriptor.nonEmpty === true}
        >
            {(binding) => {
                if (kind === "bool") {
                    return (
                        <Switch
                            id={binding.id}
                            checked={value === true}
                            disabled={loading || write.isPending}
                            onChange={(event) => {
                                write.mutate({ key: descriptor.key, value: event.target.checked });
                            }}
                        />
                    );
                }
                if (kind === "enum") {
                    const options = (descriptor.enum ?? []).map((member) => ({
                        value: member,
                        label: enumLabels[descriptor.key]?.[member] ?? member,
                    }));
                    return (
                        <Select
                            id={binding.id}
                            aria-describedby={binding["aria-describedby"]}
                            value={typeof value === "string" ? value : null}
                            options={options}
                            onValueChange={(picked) => {
                                write.mutate({ key: descriptor.key, value: picked });
                            }}
                        />
                    );
                }
                return (
                    <Input
                        id={binding.id}
                        aria-describedby={binding["aria-describedby"]}
                        invalid={binding.invalid}
                        mono={true}
                        inputMode={kind === "int" ? "numeric" : "text"}
                        value={shown}
                        disabled={loading}
                        onChange={(event) => {
                            setDraft(event.target.value);
                            setProblem(null);
                        }}
                        onBlur={(event) => {
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
            }}
        </Field>
    );

    return (
        <div className="flex flex-col gap-1.5 border-b border-hairline px-3 py-2.5 last:border-b-0">
            <div className="flex items-start gap-3">
                <div className="min-w-0 flex-1">{control}</div>
                <div className="flex w-24 shrink-0 items-center justify-end gap-1 pt-5">
                    {isDefault ? null : (
                        <>
                            <StatusBadge tone="accent" dot={false}>
                                {copy.settings.changed}
                            </StatusBadge>
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
            {prose === null ? (
                <Banner tone="warn" title={copy.settings.undocumented(descriptor.key)} />
            ) : (
                <p className="font-mono text-2xs text-ink-faint">{descriptor.key}</p>
            )}
            {prose === null ? null : <p className="text-xs text-ink-dim">{prose.help}</p>}
        </div>
    );
}
