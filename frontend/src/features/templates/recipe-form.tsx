import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { Banner, Panel, PanelHeader, Switch } from "../../ui/index.js";
import { fieldErrorOf } from "./controls.js";
import { isKnownStep, stepLabel } from "./labels.js";
import { paths } from "./patch.js";
import type { LayerView } from "./provenance.js";
import { LayerField } from "./provenance.js";
import type { SpecDraft } from "./spec.js";

export interface RecipeFormProps {
    draft: SpecDraft;
    layers: LayerView;
    error: unknown;
    onChange: (patch: Partial<SpecDraft>) => void;
}

export function RecipeForm({ draft, layers, error, onChange }: RecipeFormProps): ReactElement {
    const on = draft.recipe.filter((step) => step.enabled).length;
    const failure = fieldErrorOf(error, "recipe");

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
                                className="flex items-center gap-3 border-b border-inset py-1.5 last:border-b-0"
                            >
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
                                    {step.params === null ? null : (
                                        <span className="text-2xs text-ink-dim">
                                            {copy.templates.recipe.carriesParams}
                                        </span>
                                    )}
                                </div>
                                <Switch
                                    checked={step.enabled}
                                    aria-label={stepLabel(step.name)}
                                    onChange={(event) => {
                                        const enabled = event.target.checked;
                                        onChange({
                                            recipe: draft.recipe.map((held) =>
                                                held.name === step.name ? { ...held, enabled } : held,
                                            ),
                                        });
                                    }}
                                />
                            </li>
                        ))}
                    </ol>
                </LayerField>
            </div>
        </Panel>
    );
}
