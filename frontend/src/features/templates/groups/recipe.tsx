import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import type { JsonValue } from "../../../domain/merge-patch.js";
import { Banner, cx, IconButton, RestartAltIcon, Switch, WarningIcon } from "../../../ui/index.js";
import { fieldErrorOf, NumberInput } from "../controls.js";
import { isKnownStep, stepLabel } from "../labels.js";
import type { SpecDraft, StepDraft } from "../spec.js";

const minIterations = 1;
const maxIterations = 5;
const defaultIterations = 2;

interface SettingProps {
    step: StepDraft;
    onParam: (key: string, value: JsonValue) => void;
}

function Setting({ step, onParam }: SettingProps): ReactElement | null {
    const said = copy.templates.recipe.params;
    if (step.name === "validate") {
        return (
            <span className="flex shrink-0 items-center gap-2" title={said.allowErrorsHint}>
                <span className="text-2xs text-ink-faint">{said.allowErrors}</span>
                <Switch
                    checked={step.params?.["allowErrors"] === true}
                    disabled={!step.enabled}
                    aria-label={said.allowErrors}
                    onChange={(event) => {
                        onParam("allowErrors", event.target.checked);
                    }}
                />
            </span>
        );
    }
    if (step.name === "repair_links") {
        const held = step.params?.["iterations"];
        return (
            <span className="flex shrink-0 items-center gap-2" title={said.iterationsHint}>
                <span className="text-2xs text-ink-faint">{said.iterations}</span>
                <span className="w-16">
                    <NumberInput
                        value={typeof held === "number" ? held : defaultIterations}
                        min={minIterations}
                        max={maxIterations}
                        onValueChange={(next) => {
                            onParam(
                                "iterations",
                                Math.min(Math.max(Math.round(next), minIterations), maxIterations),
                            );
                        }}
                    />
                </span>
            </span>
        );
    }
    return null;
}

function same(a: StepDraft | undefined, b: StepDraft): boolean {
    if (a === undefined) {
        return false;
    }
    return a.enabled === b.enabled && JSON.stringify(a.params) === JSON.stringify(b.params);
}

export interface RecipeGroupProps {
    draft: SpecDraft;
    below: SpecDraft;
    error: unknown;
    onChange: (patch: Partial<SpecDraft>) => void;
}

export function RecipeGroup({ draft, below, error, onChange }: RecipeGroupProps): ReactElement {
    const on = draft.recipe.filter((step) => step.enabled).length;
    const failure = fieldErrorOf(error, "recipe");

    const patchStep = (name: string, change: (held: StepDraft) => StepDraft): void => {
        onChange({ recipe: draft.recipe.map((held) => (held.name === name ? change(held) : held)) });
    };

    return (
        <div className="flex flex-col gap-3">
            {failure === null ? null : <Banner tone="danger" title={failure} />}
            {on === 0 ? <Banner tone="warn" title={copy.templates.recipe.noneEnabled} /> : null}
            <section className="overflow-hidden rounded-lg border border-hairline bg-panel">
                <header className="flex h-7 items-center justify-between gap-3 border-b border-hairline bg-inset px-3">
                    <h3 className="text-2xs font-semibold tracking-label text-ink-faint uppercase">
                        {copy.templates.recipe.title}
                    </h3>
                    <span className="font-mono text-2xs text-ink-faint" title={copy.templates.recipe.order}>
                        {copy.templates.recipe.enabled(on, draft.recipe.length)}
                    </span>
                </header>
                {draft.recipe.map((step, index) => {
                    const under = below.recipe.find((held) => held.name === step.name);
                    const changed = !same(under, step);
                    return (
                        <div
                            key={step.name}
                            className={cx(
                                "flex h-9 items-center gap-3 border-b border-inset px-3 last:border-b-0",
                                changed && "shadow-[inset_2px_0_0_var(--color-accent)]",
                            )}
                        >
                            <span className="w-5 shrink-0 text-right font-mono text-2xs text-ink-faint">
                                {index + 1}
                            </span>
                            <span className="min-w-0 flex-1 truncate text-xs text-ink-soft">
                                {stepLabel(step.name)}
                            </span>
                            {isKnownStep(step.name) ? null : (
                                <span className="shrink-0 text-warn" title={copy.templates.recipe.unknown}>
                                    <WarningIcon size={14} />
                                </span>
                            )}
                            <Setting
                                step={step}
                                onParam={(key, value) => {
                                    patchStep(step.name, (held) => ({
                                        ...held,
                                        declared: true,
                                        params: { ...(held.params ?? {}), [key]: value },
                                    }));
                                }}
                            />
                            <Switch
                                checked={step.enabled}
                                aria-label={stepLabel(step.name)}
                                onChange={(event) => {
                                    patchStep(step.name, (held) => ({
                                        ...held,
                                        declared: true,
                                        enabled: event.target.checked,
                                    }));
                                }}
                            />
                            {changed && under !== undefined ? (
                                <IconButton
                                    icon={RestartAltIcon}
                                    label={copy.templates.overrides.revert}
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => {
                                        patchStep(step.name, () => ({ ...under }));
                                    }}
                                />
                            ) : null}
                        </div>
                    );
                })}
            </section>
        </div>
    );
}
