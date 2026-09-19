import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { JsonObject, JsonValue } from "../../domain/merge-patch.js";
import { Banner, Field, Panel, PanelHeader, Switch } from "../../ui/index.js";
import { fieldErrorOf, NumberInput } from "./controls.js";
import { isKnownStep, stepLabel } from "./labels.js";
import { paths } from "./patch.js";
import type { LayerView } from "./provenance.js";
import { LayerField } from "./provenance.js";
import type { SpecDraft, StepDraft } from "./spec.js";

export interface RecipeFormProps {
    draft: SpecDraft;
    layers: LayerView;
    error: unknown;
    onChange: (patch: Partial<SpecDraft>) => void;
}

const knownParams: Readonly<Record<string, readonly string[]>> = {
    validate: ["allowErrors"],
    repair_links: ["iterations"],
};

const minIterations = 1;
const maxIterations = 5;

function hasUnknownParams(step: StepDraft): boolean {
    if (step.params === null) {
        return false;
    }
    const known = knownParams[step.name] ?? [];
    return Object.keys(step.params).some((key) => !known.includes(key));
}

interface StepParamsProps {
    step: StepDraft;
    onParam: (key: string, value: JsonValue) => void;
}

function StepParams({ step, onParam }: StepParamsProps): ReactElement | null {
    const said = copy.templates.recipe.params;
    switch (step.name) {
        case "validate":
            return (
                <Field label={said.allowErrors} hint={said.allowErrorsHint}>
                    {(control) => (
                        <Switch
                            id={control.id}
                            aria-describedby={control["aria-describedby"]}
                            checked={step.params?.["allowErrors"] === true}
                            disabled={!step.enabled}
                            onChange={(event) => {
                                onParam("allowErrors", event.target.checked);
                            }}
                        />
                    )}
                </Field>
            );
        case "repair_links": {
            const held = step.params?.["iterations"];
            return (
                <Field label={said.iterations} hint={said.iterationsHint}>
                    {(control) => (
                        <NumberInput
                            id={control.id}
                            describedBy={control["aria-describedby"]}
                            value={typeof held === "number" ? held : 2}
                            min={minIterations}
                            max={maxIterations}
                            className="w-20"
                            onValueChange={(next) => {
                                onParam("iterations", Math.min(Math.max(Math.round(next), minIterations), maxIterations));
                            }}
                        />
                    )}
                </Field>
            );
        }
        default:
            return null;
    }
}

export function RecipeForm({ draft, layers, error, onChange }: RecipeFormProps): ReactElement {
    const on = draft.recipe.filter((step) => step.enabled).length;
    const failure = fieldErrorOf(error, "recipe");

    const patchStep = (name: string, change: (held: StepDraft) => StepDraft): void => {
        onChange({ recipe: draft.recipe.map((held) => (held.name === name ? change(held) : held)) });
    };

    return (
        <Panel className="min-w-0">
            <PanelHeader title={copy.templates.recipe.title}>
                <span className="font-mono text-2xs text-ink-faint">
                    {copy.templates.recipe.enabled(on, draft.recipe.length)}
                </span>
            </PanelHeader>
            <div className="flex flex-col gap-3 p-3">
                <p className="text-xs text-ink-dim">{copy.templates.recipe.body}</p>
                <p className="text-2xs text-ink-faint">{copy.templates.recipe.order}</p>
                {on === 0 ? <Banner tone="warn" title={copy.templates.recipe.noneEnabled} /> : null}
                {failure === null ? null : <Banner tone="danger" title={failure} />}
                <LayerField
                    layers={layers}
                    path={paths.recipe}
                    templateValue={copy.templates.recipe.enabled(
                        layers.base.recipe.filter((step) => step.enabled).length,
                        layers.base.recipe.length,
                    )}
                    onFollow={() => {
                        onChange({ recipe: layers.base.recipe.map((step) => ({ ...step })) });
                    }}
                >
                    <ol className="flex flex-col">
                        {draft.recipe.map((step, index) => (
                            <li
                                key={step.name}
                                className="flex flex-col gap-2 border-b border-inset py-1.5 last:border-b-0"
                            >
                                <div className="flex items-center gap-3">
                                    <span className="w-5 shrink-0 text-right font-mono text-2xs text-ink-faint">
                                        {index + 1}
                                    </span>
                                    <div className="flex min-w-0 flex-1 flex-col">
                                        <span className={step.enabled ? "text-sm text-ink" : "text-sm text-ink-faint"}>
                                            {stepLabel(step.name)}
                                        </span>
                                        <span className="truncate font-mono text-2xs text-ink-faint">{step.name}</span>
                                        {isKnownStep(step.name) ? null : (
                                            <span className="text-2xs text-warn">{copy.templates.recipe.unknown}</span>
                                        )}
                                        {hasUnknownParams(step) ? (
                                            <span className="text-2xs text-ink-dim">
                                                {copy.templates.recipe.carriesParams}
                                            </span>
                                        ) : null}
                                    </div>
                                    <Switch
                                        checked={step.enabled}
                                        aria-label={stepLabel(step.name)}
                                        onChange={(event) => {
                                            const enabled = event.target.checked;
                                            patchStep(step.name, (held) => ({ ...held, enabled }));
                                        }}
                                    />
                                </div>
                                {knownParams[step.name] === undefined ? null : (
                                    <div className="pl-8">
                                        <StepParams
                                            step={step}
                                            onParam={(key, value) => {
                                                patchStep(step.name, (held) => {
                                                    const params: JsonObject = { ...(held.params ?? {}), [key]: value };
                                                    return { ...held, params };
                                                });
                                            }}
                                        />
                                    </div>
                                )}
                            </li>
                        ))}
                    </ol>
                </LayerField>
            </div>
        </Panel>
    );
}
