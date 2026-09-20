import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { imageSources } from "../../generated/vocab.js";
import type { SelectOption } from "../../ui/index.js";
import { Banner, Field, Input, LinkIcon, Panel, PanelHeader, SectionLabel, Select, Switch, Textarea } from "../../ui/index.js";
import { fieldErrorOf, NumberInput } from "./controls.js";
import { decimal, flagLabel, imageSourceLabel, percent } from "./labels.js";
import { paths } from "./patch.js";
import type { LayerView } from "./provenance.js";
import { LayerField } from "./provenance.js";
import type { SpecDraft } from "./spec.js";

const unsetSource = "none";

const sourceOptions: readonly SelectOption<string>[] = [
    { value: unsetSource, label: copy.templates.meta.noSource },
    ...imageSources.map((value) => ({ value, label: imageSourceLabel(value) })),
];

export interface RulesFormProps {
    draft: SpecDraft;
    layers: LayerView;
    error: unknown;
    onChange: (patch: Partial<SpecDraft>) => void;
}

export function ContentForm({ draft, layers, error, onChange }: RulesFormProps): ReactElement {
    const base = layers.base;
    return (
        <Panel className="min-w-0">
            <PanelHeader title={copy.templates.content.title} />
            <div className="flex flex-col gap-3 p-3">
                <LayerField
                    layers={layers}
                    path={paths.tone}
                    templateValue={base.tone === "" ? copy.templates.layer.none : base.tone}
                    onFollow={() => {
                        onChange({ tone: base.tone });
                    }}
                >
                    <Field
                        label={copy.templates.content.tone}
                        hint={copy.templates.content.toneHint}
                        error={fieldErrorOf(error, "tone")}
                    >
                        {(control) => (
                            <Textarea
                                id={control.id}
                                aria-describedby={control["aria-describedby"]}
                                invalid={control.invalid}
                                rows={3}
                                value={draft.tone}
                                onChange={(event) => {
                                    onChange({ tone: event.target.value });
                                }}
                            />
                        )}
                    </Field>
                </LayerField>
                <div className="grid grid-cols-2 gap-3">
                    <LayerField
                        layers={layers}
                        path={paths.lengthMin}
                        templateValue={String(base.lengthMin)}
                        onFollow={() => {
                            onChange({ lengthMin: base.lengthMin });
                        }}
                    >
                        <Field
                            label={copy.templates.content.lengthMin}
                            hint={copy.templates.content.lengthHint}
                            error={fieldErrorOf(error, "length.min")}
                        >
                            {(control) => (
                                <NumberInput
                                    id={control.id}
                                    describedBy={control["aria-describedby"]}
                                    invalid={control.invalid}
                                    min={0}
                                    value={draft.lengthMin}
                                    onValueChange={(lengthMin) => {
                                        onChange({ lengthMin });
                                    }}
                                />
                            )}
                        </Field>
                    </LayerField>
                    <LayerField
                        layers={layers}
                        path={paths.lengthMax}
                        templateValue={String(base.lengthMax)}
                        onFollow={() => {
                            onChange({ lengthMax: base.lengthMax });
                        }}
                    >
                        <Field label={copy.templates.content.lengthMax} error={fieldErrorOf(error, "length.max")}>
                            {(control) => (
                                <NumberInput
                                    id={control.id}
                                    describedBy={control["aria-describedby"]}
                                    invalid={control.invalid}
                                    min={0}
                                    value={draft.lengthMax}
                                    onValueChange={(lengthMax) => {
                                        onChange({ lengthMax });
                                    }}
                                />
                            )}
                        </Field>
                    </LayerField>
                </div>
                <div className="flex flex-col gap-2 border-t border-hairline pt-3">
                    <SectionLabel>{copy.templates.content.keywords}</SectionLabel>
                    <p className="text-xs text-ink-dim">{copy.templates.content.keywordsBody}</p>
                    <LayerField
                        layers={layers}
                        path={paths.primaryInTitle}
                        templateValue={flagLabel(base.primaryInTitle)}
                        onFollow={() => {
                            onChange({ primaryInTitle: base.primaryInTitle });
                        }}
                    >
                        <Switch
                            className="w-full"
                            label={copy.templates.content.primaryInTitle}
                            checked={draft.primaryInTitle}
                            onChange={(event) => {
                                onChange({ primaryInTitle: event.target.checked });
                            }}
                        />
                    </LayerField>
                    <LayerField
                        layers={layers}
                        path={paths.primaryInH1}
                        templateValue={flagLabel(base.primaryInH1)}
                        onFollow={() => {
                            onChange({ primaryInH1: base.primaryInH1 });
                        }}
                    >
                        <Switch
                            className="w-full"
                            label={copy.templates.content.primaryInH1}
                            checked={draft.primaryInH1}
                            onChange={(event) => {
                                onChange({ primaryInH1: event.target.checked });
                            }}
                        />
                    </LayerField>
                    <LayerField
                        layers={layers}
                        path={paths.primaryInFirstParagraph}
                        templateValue={flagLabel(base.primaryInFirstParagraph)}
                        onFollow={() => {
                            onChange({ primaryInFirstParagraph: base.primaryInFirstParagraph });
                        }}
                    >
                        <Switch
                            className="w-full"
                            label={copy.templates.content.primaryInFirstParagraph}
                            checked={draft.primaryInFirstParagraph}
                            onChange={(event) => {
                                onChange({ primaryInFirstParagraph: event.target.checked });
                            }}
                        />
                    </LayerField>
                    <LayerField
                        layers={layers}
                        path={paths.maxDensity}
                        templateValue={percent(base.maxDensity)}
                        onFollow={() => {
                            onChange({ maxDensity: base.maxDensity });
                        }}
                    >
                        <Field
                            label={copy.templates.content.maxDensity}
                            hint={copy.templates.content.maxDensityHint}
                            error={fieldErrorOf(error, "keywordRules.maxDensity")}
                        >
                            {(control) => (
                                <NumberInput
                                    id={control.id}
                                    describedBy={control["aria-describedby"]}
                                    invalid={control.invalid}
                                    min={0}
                                    max={1}
                                    step={0.005}
                                    value={draft.maxDensity}
                                    onValueChange={(maxDensity) => {
                                        onChange({ maxDensity });
                                    }}
                                />
                            )}
                        </Field>
                    </LayerField>
                </div>
            </div>
        </Panel>
    );
}

export function LinksForm({ draft, layers, error, onChange }: RulesFormProps): ReactElement {
    const base = layers.base;
    return (
        <Panel className="min-w-0">
            <PanelHeader title={copy.templates.links.title} />
            <div className="flex flex-col gap-3 p-3">
                <p className="text-xs text-ink-dim">{copy.templates.links.body}</p>
                <Banner tone="info" icon={LinkIcon} title={copy.templates.links.fromTemplate} />
                <div className="grid grid-cols-2 gap-3">
                    <LayerField
                        layers={layers}
                        path={paths.upDepth}
                        templateValue={String(base.upDepth)}
                        onFollow={() => {
                            onChange({ upDepth: base.upDepth });
                        }}
                    >
                        <Field
                            label={copy.templates.links.upDepth}
                            hint={copy.templates.links.upDepthHint}
                            error={fieldErrorOf(error, "linkRules.upDepth")}
                        >
                            {(control) => (
                                <NumberInput
                                    id={control.id}
                                    describedBy={control["aria-describedby"]}
                                    invalid={control.invalid}
                                    min={0}
                                    value={draft.upDepth}
                                    onValueChange={(upDepth) => {
                                        onChange({ upDepth });
                                    }}
                                />
                            )}
                        </Field>
                    </LayerField>
                    <LayerField
                        layers={layers}
                        path={paths.siblingMinWeight}
                        templateValue={decimal(base.siblingMinWeight)}
                        onFollow={() => {
                            onChange({ siblingMinWeight: base.siblingMinWeight });
                        }}
                    >
                        <Field
                            label={copy.templates.links.siblingMinWeight}
                            hint={copy.templates.links.siblingMinWeightHint}
                            error={fieldErrorOf(error, "linkRules.siblingMinWeight")}
                        >
                            {(control) => (
                                <NumberInput
                                    id={control.id}
                                    describedBy={control["aria-describedby"]}
                                    invalid={control.invalid}
                                    min={0}
                                    max={1}
                                    step={0.05}
                                    value={draft.siblingMinWeight}
                                    onValueChange={(siblingMinWeight) => {
                                        onChange({ siblingMinWeight });
                                    }}
                                />
                            )}
                        </Field>
                    </LayerField>
                    <LayerField
                        layers={layers}
                        path={paths.maxLinks}
                        templateValue={String(base.maxLinks)}
                        onFollow={() => {
                            onChange({ maxLinks: base.maxLinks });
                        }}
                    >
                        <Field
                            label={copy.templates.links.maxLinks}
                            error={fieldErrorOf(error, "linkRules.maxLinks")}
                        >
                            {(control) => (
                                <NumberInput
                                    id={control.id}
                                    describedBy={control["aria-describedby"]}
                                    invalid={control.invalid}
                                    min={0}
                                    value={draft.maxLinks}
                                    onValueChange={(maxLinks) => {
                                        onChange({ maxLinks });
                                    }}
                                />
                            )}
                        </Field>
                    </LayerField>
                    <LayerField
                        layers={layers}
                        path={paths.maxPerTarget}
                        templateValue={String(base.maxPerTarget)}
                        onFollow={() => {
                            onChange({ maxPerTarget: base.maxPerTarget });
                        }}
                    >
                        <Field
                            label={copy.templates.links.maxPerTarget}
                            error={fieldErrorOf(error, "linkRules.maxPerTarget")}
                        >
                            {(control) => (
                                <NumberInput
                                    id={control.id}
                                    describedBy={control["aria-describedby"]}
                                    invalid={control.invalid}
                                    min={0}
                                    value={draft.maxPerTarget}
                                    onValueChange={(maxPerTarget) => {
                                        onChange({ maxPerTarget });
                                    }}
                                />
                            )}
                        </Field>
                    </LayerField>
                    <LayerField
                        layers={layers}
                        path={paths.parentLinkWithinParagraphs}
                        templateValue={String(base.parentLinkWithinParagraphs)}
                        onFollow={() => {
                            onChange({ parentLinkWithinParagraphs: base.parentLinkWithinParagraphs });
                        }}
                    >
                        <Field
                            label={copy.templates.links.parentLinkWithinParagraphs}
                            hint={copy.templates.links.parentLinkWithinParagraphsHint}
                            error={fieldErrorOf(error, "linkRules.parentLinkWithinParagraphs")}
                        >
                            {(control) => (
                                <NumberInput
                                    id={control.id}
                                    describedBy={control["aria-describedby"]}
                                    invalid={control.invalid}
                                    min={0}
                                    value={draft.parentLinkWithinParagraphs}
                                    onValueChange={(parentLinkWithinParagraphs) => {
                                        onChange({ parentLinkWithinParagraphs });
                                    }}
                                />
                            )}
                        </Field>
                    </LayerField>
                </div>
                <div className="flex flex-col gap-2 border-t border-hairline pt-3">
                    <LayerField
                        layers={layers}
                        path={paths.downLinks}
                        templateValue={flagLabel(base.downLinks)}
                        onFollow={() => {
                            onChange({ downLinks: base.downLinks });
                        }}
                    >
                        <Switch
                            className="w-full"
                            label={copy.templates.links.downLinks}
                            checked={draft.downLinks}
                            onChange={(event) => {
                                onChange({ downLinks: event.target.checked });
                            }}
                        />
                    </LayerField>
                    <LayerField
                        layers={layers}
                        path={paths.childrenSection}
                        templateValue={flagLabel(base.childrenSection)}
                        onFollow={() => {
                            onChange({ childrenSection: base.childrenSection });
                        }}
                    >
                        <Switch
                            className="w-full"
                            label={copy.templates.links.childrenSection}
                            checked={draft.childrenSection}
                            onChange={(event) => {
                                onChange({ childrenSection: event.target.checked });
                            }}
                        />
                    </LayerField>
                </div>
            </div>
        </Panel>
    );
}

export function MetaForm({ draft, layers, error, onChange }: RulesFormProps): ReactElement {
    const base = layers.base;
    return (
        <Panel className="min-w-0">
            <PanelHeader title={copy.templates.meta.title} />
            <div className="flex flex-col gap-3 p-3">
                <LayerField
                    layers={layers}
                    path={paths.titlePattern}
                    templateValue={base.titlePattern === "" ? copy.templates.layer.none : base.titlePattern}
                    onFollow={() => {
                        onChange({ titlePattern: base.titlePattern });
                    }}
                >
                    <Field
                        label={copy.templates.meta.titlePattern}
                        hint={copy.templates.meta.titlePatternHint}
                        error={fieldErrorOf(error, "metaRules.titlePattern")}
                    >
                        {(control) => (
                            <Input
                                id={control.id}
                                aria-describedby={control["aria-describedby"]}
                                invalid={control.invalid}
                                mono={true}
                                value={draft.titlePattern}
                                onChange={(event) => {
                                    onChange({ titlePattern: event.target.value });
                                }}
                            />
                        )}
                    </Field>
                </LayerField>
                <LayerField
                    layers={layers}
                    path={paths.descriptionMax}
                    templateValue={String(base.descriptionMax)}
                    onFollow={() => {
                        onChange({ descriptionMax: base.descriptionMax });
                    }}
                >
                    <Field
                        label={copy.templates.meta.descriptionMax}
                        hint={copy.templates.meta.descriptionMaxHint}
                        error={fieldErrorOf(error, "metaRules.descriptionMax")}
                    >
                        {(control) => (
                            <NumberInput
                                id={control.id}
                                describedBy={control["aria-describedby"]}
                                invalid={control.invalid}
                                min={0}
                                value={draft.descriptionMax}
                                onValueChange={(descriptionMax) => {
                                    onChange({ descriptionMax });
                                }}
                            />
                        )}
                    </Field>
                </LayerField>
                <div className="flex flex-col gap-2.5 border-t border-hairline pt-3">
                    <SectionLabel>{copy.templates.meta.images}</SectionLabel>
                    <LayerField
                        layers={layers}
                        path={paths.featuredImage}
                        templateValue={flagLabel(base.featuredImage)}
                        onFollow={() => {
                            onChange({ featuredImage: base.featuredImage });
                        }}
                    >
                        <Switch
                            className="w-full"
                            label={copy.templates.meta.featured}
                            checked={draft.featuredImage}
                            onChange={(event) => {
                                onChange({ featuredImage: event.target.checked });
                            }}
                        />
                    </LayerField>
                    <div className="grid grid-cols-2 gap-3">
                        <LayerField
                            layers={layers}
                            path={paths.inlineImages}
                            templateValue={String(base.inlineImages)}
                            onFollow={() => {
                                onChange({ inlineImages: base.inlineImages });
                            }}
                        >
                            <Field
                                label={copy.templates.meta.inline}
                                error={fieldErrorOf(error, "images.inline")}
                            >
                                {(control) => (
                                    <NumberInput
                                        id={control.id}
                                        describedBy={control["aria-describedby"]}
                                        invalid={control.invalid}
                                        min={0}
                                        value={draft.inlineImages}
                                        onValueChange={(inlineImages) => {
                                            onChange({ inlineImages });
                                        }}
                                    />
                                )}
                            </Field>
                        </LayerField>
                        <LayerField
                            layers={layers}
                            path={paths.imageSource}
                            templateValue={imageSourceLabel(base.imageSource)}
                            onFollow={() => {
                                onChange({ imageSource: base.imageSource });
                            }}
                        >
                            <Field
                                label={copy.templates.meta.source}
                                hint={copy.templates.meta.sourceHint}
                                error={fieldErrorOf(error, "images.source")}
                            >
                                {(control) => (
                                    <Select
                                        id={control.id}
                                        aria-describedby={control["aria-describedby"]}
                                        invalid={control.invalid}
                                        value={draft.imageSource === "" ? unsetSource : draft.imageSource}
                                        options={sourceOptions}
                                        onValueChange={(next) => {
                                            onChange({ imageSource: next === unsetSource ? "" : next });
                                        }}
                                    />
                                )}
                            </Field>
                        </LayerField>
                    </div>
                </div>
            </div>
        </Panel>
    );
}
