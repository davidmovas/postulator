import type { ReactElement } from "react";
import { useMemo } from "react";

import { copy } from "../../copy/index.js";
import { fieldErrorOf } from "../../data/errors.js";
import type { Entity, Schedule, Template } from "../../data/types.js";
import { Field, Input, Segmented, SectionLabel, Select, StatusBadge } from "../../ui/index.js";
import { stepLabel } from "../runs/labels.js";
import { CadenceFields } from "./cadence.js";
import type { ScheduleDraft } from "./form.js";
import { publishLabel } from "./labels.js";

export const anyStatus = "any";
export const anyEntity = "any";
export const siteTemplate = "site";

export interface ScheduleFieldsProps {
    draft: ScheduleDraft;
    schedule: Schedule | null;
    templates: readonly Template[];
    entities: readonly Entity[];
    thrown: unknown;
    onChange: (next: ScheduleDraft) => void;
}

export function ScheduleFields({
    draft,
    schedule,
    templates,
    entities,
    thrown,
    onChange,
}: ScheduleFieldsProps): ReactElement {
    const cronError = fieldErrorOf(thrown, "cron") ?? fieldErrorOf(thrown, "interval");

    const templateOptions = useMemo(
        () => [
            { value: siteTemplate, label: copy.schedules.panel.anyTemplate },
            ...templates.map((template) => ({ value: template.id, label: template.name })),
        ],
        [templates],
    );

    const entityOptions = useMemo(
        () => [
            { value: anyEntity, label: copy.schedules.panel.anyEntity },
            ...entities.map((entity) => ({ value: entity.id, label: entity.name })),
        ],
        [entities],
    );

    return (
        <div className="flex flex-col gap-3">
            <CadenceFields draft={draft} cronError={cronError} onChange={onChange} />

            <SectionLabel>{copy.schedules.panel.work}</SectionLabel>
            <Field label={copy.schedules.panel.template}>
                {(control) => (
                    <Select
                        id={control.id}
                        value={draft.templateId === "" ? siteTemplate : draft.templateId}
                        options={templateOptions}
                        onValueChange={(next) => {
                            onChange({ ...draft, templateId: next === siteTemplate ? "" : next });
                        }}
                    />
                )}
            </Field>
            <Field label={copy.schedules.panel.publish}>
                {() => (
                    <Segmented
                        label={copy.schedules.panel.publish}
                        value={draft.publishMode}
                        options={[
                            { value: "draft", label: publishLabel("draft") },
                            { value: "publish", label: publishLabel("publish") },
                        ]}
                        onValueChange={(next) => {
                            onChange({ ...draft, publishMode: next });
                        }}
                    />
                )}
            </Field>
            <div className="flex flex-col gap-1">
                <SectionLabel>{copy.schedules.panel.recipe}</SectionLabel>
                {schedule === null || schedule.steps === null || schedule.steps.length === 0 ? (
                    <p className="text-2xs text-ink-faint">{copy.schedules.panel.templateRecipe}</p>
                ) : (
                    <div className="flex flex-wrap gap-1">
                        {schedule.steps.map((step) => (
                            <StatusBadge key={step} tone="muted" dot={false}>
                                {stepLabel(step)}
                            </StatusBadge>
                        ))}
                    </div>
                )}
            </div>

            <SectionLabel>{copy.schedules.panel.targets}</SectionLabel>
            <Field label={copy.schedules.panel.status} error={fieldErrorOf(thrown, "status")}>
                {(control) => (
                    <Select
                        id={control.id}
                        value={draft.status === "" ? anyStatus : draft.status}
                        options={[
                            { value: anyStatus, label: copy.schedules.panel.anyStatus },
                            { value: "planned", label: copy.schedules.statuses.planned },
                            { value: "exists", label: copy.schedules.statuses.exists },
                            { value: "published", label: copy.schedules.statuses.published },
                            { value: "archived", label: copy.schedules.statuses.archived },
                        ]}
                        onValueChange={(next) => {
                            onChange({ ...draft, status: next === anyStatus ? "" : next });
                        }}
                    />
                )}
            </Field>
            <Field label={copy.schedules.panel.entity}>
                {(control) => (
                    <Select
                        id={control.id}
                        value={draft.entityId === "" ? anyEntity : draft.entityId}
                        options={entityOptions}
                        disabled={entities.length === 0}
                        onValueChange={(next) => {
                            onChange({ ...draft, entityId: next === anyEntity ? "" : next });
                        }}
                    />
                )}
            </Field>
            <Field label={copy.schedules.panel.limit} error={fieldErrorOf(thrown, "limit")}>
                {(control) => (
                    <Input
                        id={control.id}
                        type="number"
                        min={1}
                        max={500}
                        mono={true}
                        invalid={control.invalid}
                        value={String(draft.limit)}
                        onChange={(event) => {
                            onChange({ ...draft, limit: Number(event.target.value) });
                        }}
                    />
                )}
            </Field>

            <SectionLabel>{copy.schedules.panel.budget}</SectionLabel>
            <div className="flex gap-2">
                <Field label={copy.schedules.panel.maxUsd} error={fieldErrorOf(thrown, "budget")} className="flex-1">
                    {(control) => (
                        <Input
                            id={control.id}
                            type="number"
                            min={0}
                            step={0.5}
                            mono={true}
                            invalid={control.invalid}
                            value={draft.maxUsd === 0 ? "" : String(draft.maxUsd)}
                            placeholder={copy.schedules.panel.noCap}
                            onChange={(event) => {
                                onChange({ ...draft, maxUsd: Number(event.target.value) });
                            }}
                        />
                    )}
                </Field>
                <Field label={copy.schedules.panel.maxTokens} className="flex-1">
                    {(control) => (
                        <Input
                            id={control.id}
                            type="number"
                            min={0}
                            mono={true}
                            value={draft.maxTokens === 0 ? "" : String(draft.maxTokens)}
                            placeholder={copy.schedules.panel.noCap}
                            onChange={(event) => {
                                onChange({ ...draft, maxTokens: Number(event.target.value) });
                            }}
                        />
                    )}
                </Field>
            </div>
        </div>
    );
}
