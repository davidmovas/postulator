import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import {
    AddIcon,
    ArrowDownwardIcon,
    ArrowUpwardIcon,
    Button,
    ChipInput,
    cx,
    DeleteIcon,
    EmptyState,
    IconButton,
    Input,
    RestartAltIcon,
    Switch,
    Textarea,
    TopicIcon,
} from "../../../ui/index.js";
import { fieldErrorOf, NumberInput } from "../controls.js";
import type { SectionDraft, SpecDraft } from "../spec.js";
import { emptySection, moved } from "../spec.js";

interface CardProps {
    section: SectionDraft;
    index: number;
    total: number;
    error: unknown;
    onChange: (patch: Partial<SectionDraft>) => void;
    onMove: (to: number) => void;
    onRemove: () => void;
}

function Card({ section, index, total, error, onChange, onMove, onRemove }: CardProps): ReactElement {
    const prefix = `sections[${index}]`;
    const headingError = fieldErrorOf(error, `${prefix}.heading`);
    return (
        <section className="overflow-hidden rounded-lg border border-hairline bg-panel">
            <header className="flex h-10 items-center gap-2 border-b border-hairline px-2.5">
                <span className="w-4 shrink-0 text-center font-mono text-2xs text-ink-faint">{index + 1}</span>
                <div className="min-w-0 flex-1">
                    <Input
                        value={section.heading}
                        invalid={headingError !== null}
                        aria-label={copy.templates.sections.heading}
                        placeholder={copy.templates.sections.heading}
                        onChange={(event) => {
                            onChange({ heading: event.target.value });
                        }}
                    />
                </div>
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
            </header>
            <div className="flex flex-col gap-2 p-2.5">
                {headingError === null ? null : <p className="text-2xs text-danger">{headingError}</p>}
                <Textarea
                    rows={3}
                    value={section.intent}
                    aria-label={copy.templates.sections.intent}
                    placeholder={copy.templates.sections.intent}
                    invalid={fieldErrorOf(error, `${prefix}.intent`) !== null}
                    onChange={(event) => {
                        onChange({ intent: event.target.value });
                    }}
                />
                <div className="flex items-center gap-2">
                    <span className="shrink-0 text-2xs text-ink-faint">
                        {copy.templates.sections.targetWords}
                    </span>
                    <div className="w-20 shrink-0">
                        <NumberInput
                            value={section.targetWords}
                            min={0}
                            invalid={fieldErrorOf(error, `${prefix}.targetWords`) !== null}
                            onValueChange={(targetWords) => {
                                onChange({ targetWords });
                            }}
                        />
                    </div>
                </div>
                <div className="flex flex-wrap items-center gap-x-6 gap-y-2">
                    <span title={copy.templates.sections.requiredHint}>
                        <Switch
                            label={copy.templates.sections.required}
                            checked={section.required}
                            onChange={(event) => {
                                onChange({ required: event.target.checked });
                            }}
                        />
                    </span>
                    <Switch
                        label={copy.templates.sections.primaryInHeading}
                        checked={section.primaryInHeading}
                        onChange={(event) => {
                            onChange({ primaryInHeading: event.target.checked });
                        }}
                    />
                </div>
                <div className="flex flex-col gap-1">
                    <span className="text-2xs text-ink-faint">{copy.templates.sections.include}</span>
                    <ChipInput
                        values={section.include}
                        mono={true}
                        placeholder={copy.templates.sections.includePlaceholder}
                        removeLabel={copy.templates.sections.includeRemove}
                        onChange={(include) => {
                            onChange({ include });
                        }}
                    />
                </div>
            </div>
        </section>
    );
}

export interface SectionsGroupProps {
    draft: SpecDraft;
    below: SpecDraft;
    error: unknown;
    onChange: (patch: Partial<SpecDraft>) => void;
}

export function SectionsGroup({ draft, below, error, onChange }: SectionsGroupProps): ReactElement {
    const total = draft.sections.length;
    const changed = JSON.stringify(draft.sections) !== JSON.stringify(below.sections);

    const replace = (index: number, patch: Partial<SectionDraft>): void => {
        onChange({
            sections: draft.sections.map((section, at) => (at === index ? { ...section, ...patch } : section)),
        });
    };

    if (total === 0) {
        return (
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
        );
    }

    return (
        <div className={cx("flex flex-col gap-2", changed && "pl-2 shadow-[inset_2px_0_0_var(--color-accent)]")}>
            {draft.sections.map((section, index) => (
                <Card
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
            ))}
            <div className="flex gap-2">
                <Button
                    icon={AddIcon}
                    onClick={() => {
                        onChange({ sections: [...draft.sections, emptySection()] });
                    }}
                >
                    {copy.templates.sections.add}
                </Button>
                {changed ? (
                    <Button
                        icon={RestartAltIcon}
                        onClick={() => {
                            onChange({
                                sections: below.sections.map((section) => ({
                                    ...section,
                                    include: [...section.include],
                                })),
                            });
                        }}
                    >
                        {copy.templates.overrides.revert}
                    </Button>
                ) : null}
            </div>
        </div>
    );
}
