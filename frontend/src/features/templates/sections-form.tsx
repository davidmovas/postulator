import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../copy/index.js";
import {
    AddIcon,
    ArrowDownwardIcon,
    ArrowUpwardIcon,
    Button,
    Checkbox,
    CloseIcon,
    DeleteIcon,
    EmptyState,
    Field,
    IconButton,
    Input,
    Panel,
    PanelHeader,
    Switch,
    Textarea,
    TopicIcon,
} from "../../ui/index.js";
import { fieldErrorOf, NumberInput } from "./controls.js";
import { paths } from "./patch.js";
import type { LayerView } from "./provenance.js";
import { LayerField } from "./provenance.js";
import type { SectionDraft, SpecDraft } from "./spec.js";
import { emptySection, moved } from "./spec.js";

interface KeywordChipsProps {
    words: readonly string[];
    onChange: (words: string[]) => void;
}

function KeywordChips({ words, onChange }: KeywordChipsProps): ReactElement {
    const [pending, setPending] = useState("");

    const add = (): void => {
        const word = pending.trim();
        if (word === "" || words.includes(word)) {
            setPending("");
            return;
        }
        onChange([...words, word]);
        setPending("");
    };

    return (
        <div className="flex flex-col gap-1.5">
            {words.length === 0 ? (
                <p className="text-2xs text-ink-faint">{copy.templates.sections.includeEmpty}</p>
            ) : (
                <div className="flex flex-wrap gap-1">
                    {words.map((word) => (
                        <span
                            key={word}
                            className="inline-flex h-5 items-center gap-1 rounded-sm border border-hairline bg-inset px-1.5 font-mono text-2xs text-ink-soft"
                        >
                            {word}
                            <button
                                type="button"
                                aria-label={copy.templates.sections.includeRemove(word)}
                                title={copy.templates.sections.includeRemove(word)}
                                onClick={() => {
                                    onChange(words.filter((held) => held !== word));
                                }}
                                className="text-ink-faint hover:text-danger"
                            >
                                <CloseIcon size={11} />
                            </button>
                        </span>
                    ))}
                </div>
            )}
            <div className="flex gap-1.5">
                <Input
                    value={pending}
                    placeholder={copy.templates.sections.includePlaceholder}
                    onChange={(event) => {
                        setPending(event.target.value);
                    }}
                    onKeyDown={(event) => {
                        if (event.key === "Enter") {
                            event.preventDefault();
                            add();
                        }
                    }}
                />
                <Button size="sm" onClick={add} disabled={pending.trim() === ""}>
                    {copy.templates.sections.includeAdd}
                </Button>
            </div>
        </div>
    );
}

interface SectionCardProps {
    section: SectionDraft;
    index: number;
    total: number;
    error: unknown;
    onChange: (patch: Partial<SectionDraft>) => void;
    onMove: (to: number) => void;
    onRemove: () => void;
}

function SectionCard({ section, index, total, error, onChange, onMove, onRemove }: SectionCardProps): ReactElement {
    const prefix = `sections[${index}]`;
    return (
        <div className="flex flex-col gap-2.5 rounded-md border border-hairline bg-inset p-2.5">
            <div className="flex items-center justify-between gap-2">
                <span className="font-mono text-2xs text-ink-faint">
                    {copy.templates.sections.position(index + 1, total)}
                </span>
                <div className="flex shrink-0 gap-1">
                    <IconButton
                        icon={ArrowUpwardIcon}
                        label={copy.templates.sections.moveUp}
                        variant="ghost"
                        size="sm"
                        disabled={index === 0}
                        onClick={() => {
                            onMove(index - 1);
                        }}
                    />
                    <IconButton
                        icon={ArrowDownwardIcon}
                        label={copy.templates.sections.moveDown}
                        variant="ghost"
                        size="sm"
                        disabled={index === total - 1}
                        onClick={() => {
                            onMove(index + 1);
                        }}
                    />
                    <IconButton
                        icon={DeleteIcon}
                        label={copy.templates.sections.remove}
                        variant="ghost"
                        size="sm"
                        onClick={onRemove}
                    />
                </div>
            </div>
            <Field
                label={copy.templates.sections.heading}
                required={true}
                error={fieldErrorOf(error, `${prefix}.heading`)}
            >
                {(control) => (
                    <Input
                        id={control.id}
                        aria-describedby={control["aria-describedby"]}
                        invalid={control.invalid}
                        value={section.heading}
                        onChange={(event) => {
                            onChange({ heading: event.target.value });
                        }}
                    />
                )}
            </Field>
            <Field
                label={copy.templates.sections.intent}
                hint={copy.templates.sections.intentHint}
                error={fieldErrorOf(error, `${prefix}.intent`)}
            >
                {(control) => (
                    <Textarea
                        id={control.id}
                        aria-describedby={control["aria-describedby"]}
                        invalid={control.invalid}
                        rows={3}
                        value={section.intent}
                        onChange={(event) => {
                            onChange({ intent: event.target.value });
                        }}
                    />
                )}
            </Field>
            <div className="grid grid-cols-2 items-end gap-2">
                <Field
                    label={copy.templates.sections.targetWords}
                    error={fieldErrorOf(error, `${prefix}.targetWords`)}
                >
                    {(control) => (
                        <NumberInput
                            id={control.id}
                            describedBy={control["aria-describedby"]}
                            invalid={control.invalid}
                            min={0}
                            value={section.targetWords}
                            onValueChange={(targetWords) => {
                                onChange({ targetWords });
                            }}
                        />
                    )}
                </Field>
                <Switch
                    label={copy.templates.sections.required}
                    checked={section.required}
                    onChange={(event) => {
                        onChange({ required: event.target.checked });
                    }}
                />
            </div>
            <Checkbox
                label={copy.templates.sections.primaryInHeading}
                checked={section.primaryInHeading}
                onChange={(event) => {
                    onChange({ primaryInHeading: event.target.checked });
                }}
            />
            <div className="flex flex-col gap-1">
                <span className="text-xs font-medium text-ink-soft">{copy.templates.sections.include}</span>
                <KeywordChips
                    words={section.include}
                    onChange={(include) => {
                        onChange({ include });
                    }}
                />
            </div>
        </div>
    );
}

export interface SectionsFormProps {
    draft: SpecDraft;
    layers: LayerView;
    error: unknown;
    onChange: (patch: Partial<SpecDraft>) => void;
}

export function SectionsForm({ draft, layers, error, onChange }: SectionsFormProps): ReactElement {
    const total = draft.sections.length;

    const replace = (index: number, patch: Partial<SectionDraft>): void => {
        onChange({
            sections: draft.sections.map((section, at) => (at === index ? { ...section, ...patch } : section)),
        });
    };

    return (
        <Panel className="min-w-0">
            <PanelHeader title={copy.templates.sections.title}>
                <Button
                    size="sm"
                    icon={AddIcon}
                    onClick={() => {
                        onChange({ sections: [...draft.sections, emptySection()] });
                    }}
                >
                    {copy.templates.sections.add}
                </Button>
            </PanelHeader>
            <div className="flex flex-col gap-2.5 p-3">
                <p className="text-xs text-ink-dim">{copy.templates.sections.body}</p>
                <LayerField
                    layers={layers}
                    path={paths.sections}
                    templateValue={copy.templates.sectionCount(layers.base.sections.length)}
                    onFollow={() => {
                        onChange({ sections: layers.base.sections.map((section) => ({ ...section })) });
                    }}
                >
                    <div className="flex flex-col gap-2.5">
                        {total === 0 ? (
                            <EmptyState
                                icon={TopicIcon}
                                title={copy.templates.sections.empty}
                                body={copy.templates.sections.emptyBody}
                                actions={
                                    <Button
                                        variant="primary"
                                        icon={AddIcon}
                                        onClick={() => {
                                            onChange({ sections: [emptySection()] });
                                        }}
                                    >
                                        {copy.templates.sections.add}
                                    </Button>
                                }
                            />
                        ) : (
                            draft.sections.map((section, index) => (
                                <SectionCard
                                    key={index}
                                    section={section}
                                    index={index}
                                    total={total}
                                    error={error}
                                    onChange={(patch) => {
                                        replace(index, patch);
                                    }}
                                    onMove={(to) => {
                                        onChange({ sections: moved(draft.sections, index, to) });
                                    }}
                                    onRemove={() => {
                                        onChange({ sections: draft.sections.filter((_held, at) => at !== index) });
                                    }}
                                />
                            ))
                        )}
                    </div>
                </LayerField>
            </div>
        </Panel>
    );
}
