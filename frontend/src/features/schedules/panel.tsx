import type { ReactElement } from "react";
import { useEffect, useMemo, useState } from "react";

import { copy } from "../../copy/index.js";
import { fieldErrorOf, formErrorOf } from "../../data/errors.js";
import {
    useCreateSchedule,
    useDeleteSchedule,
    useDisableSchedule,
    useEnableSchedule,
    useRunScheduleNow,
    useUpdateSchedule,
} from "../../data/hooks/schedules.js";
import type { Entity, Schedule, Template } from "../../data/types.js";
import {
    Banner,
    Button,
    CloseIcon,
    Dialog,
    Field,
    IconButton,
    Input,
    PanelHeader,
    PlayArrowIcon,
    Segmented,
    SectionLabel,
    Select,
    StatusBadge,
} from "../../ui/index.js";
import { stepLabel } from "../runs/labels.js";
import { CadenceFields } from "./cadence.js";
import { createOf, draftOf, dirty, ready, updateOf } from "./form.js";
import type { ScheduleDraft } from "./form.js";
import { actorLabel, publishLabel } from "./labels.js";
import { LastRunCard } from "./last-run.js";

const anyStatus = "any";
const anyEntity = "any";
const siteTemplate = "site";

export interface SchedulePanelProps {
    siteId: string;
    schedule: Schedule | null;
    templates: readonly Template[];
    entities: readonly Entity[];
    onClose: () => void;
    onCreated: (schedule: Schedule) => void;
    onStarted: (runId: string) => void;
    onDeleted: () => void;
}

export function SchedulePanel({
    siteId,
    schedule,
    templates,
    entities,
    onClose,
    onCreated,
    onStarted,
    onDeleted,
}: SchedulePanelProps): ReactElement {
    const [draft, setDraft] = useState<ScheduleDraft>(() => draftOf(schedule));
    const [doomed, setDoomed] = useState(false);
    const create = useCreateSchedule();
    const update = useUpdateSchedule();
    const remove = useDeleteSchedule();
    const enable = useEnableSchedule();
    const disable = useDisableSchedule();
    const runNow = useRunScheduleNow();

    useEffect(() => {
        setDraft(draftOf(schedule));
    }, [schedule]);

    const thrown = schedule === null ? create.error : update.error;
    const cronError = fieldErrorOf(thrown, "cron") ?? fieldErrorOf(thrown, "interval");
    const formError = formErrorOf(thrown);
    const changed = dirty(draft, schedule);
    const saving = create.isPending || update.isPending;

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

    const save = (): void => {
        if (schedule === null) {
            create.mutate(createOf(draft, siteId), {
                onSuccess: (answered) => {
                    onCreated(answered.schedule);
                },
            });
            return;
        }
        update.mutate(updateOf(draft, schedule));
    };

    return (
        <div className="flex flex-col gap-3 p-3">
            <PanelHeader title={schedule?.name ?? copy.schedules.panel.create}>
                <IconButton
                    icon={CloseIcon}
                    label={copy.schedules.panel.close}
                    size="sm"
                    variant="ghost"
                    onClick={onClose}
                />
            </PanelHeader>
            <Field label={copy.schedules.panel.name} error={fieldErrorOf(thrown, "name")} required={true}>
                {(control) => (
                    <Input
                        id={control.id}
                        value={draft.name}
                        invalid={control.invalid}
                        placeholder={copy.schedules.panel.namePlaceholder}
                        onChange={(event) => {
                            setDraft({ ...draft, name: event.target.value });
                        }}
                    />
                )}
            </Field>
            <CadenceFields draft={draft} cronError={cronError} onChange={setDraft} />

            <SectionLabel>{copy.schedules.panel.work}</SectionLabel>
            <Field label={copy.schedules.panel.template}>
                {(control) => (
                    <Select
                        id={control.id}
                        value={draft.templateId === "" ? siteTemplate : draft.templateId}
                        options={templateOptions}
                        onValueChange={(next) => {
                            setDraft({ ...draft, templateId: next === siteTemplate ? "" : next });
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
                            setDraft({ ...draft, publishMode: next });
                        }}
                    />
                )}
            </Field>
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
                            setDraft({ ...draft, status: next === anyStatus ? "" : next });
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
                            setDraft({ ...draft, entityId: next === anyEntity ? "" : next });
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
                            setDraft({ ...draft, limit: Number(event.target.value) });
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
                            value={String(draft.maxUsd)}
                            placeholder={copy.schedules.panel.noCap}
                            onChange={(event) => {
                                setDraft({ ...draft, maxUsd: Number(event.target.value) });
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
                            value={String(draft.maxTokens)}
                            placeholder={copy.schedules.panel.noCap}
                            onChange={(event) => {
                                setDraft({ ...draft, maxTokens: Number(event.target.value) });
                            }}
                        />
                    )}
                </Field>
            </div>

            {formError === null ? null : <Banner tone="danger" title={formError} />}
            {runNow.data?.skipped === undefined || runNow.data.skipped === "" ? null : (
                <Banner tone="warn" title={runNow.data.skipped} />
            )}

            <div className="flex flex-wrap gap-2">
                <Button variant="primary" busy={saving} disabled={!ready(draft) || !changed} onClick={save}>
                    {copy.schedules.panel.save}
                </Button>
                {schedule === null ? null : (
                    <>
                        <Button
                            variant="secondary"
                            icon={PlayArrowIcon}
                            busy={runNow.isPending}
                            onClick={() => {
                                runNow.mutate(
                                    { id: schedule.id },
                                    {
                                        onSuccess: (answered) => {
                                            if (answered.runId !== "") {
                                                onStarted(answered.runId);
                                            }
                                        },
                                    },
                                );
                            }}
                        >
                            {copy.schedules.panel.runNow}
                        </Button>
                        <Button
                            variant="secondary"
                            busy={enable.isPending || disable.isPending}
                            onClick={() => {
                                if (schedule.enabled) {
                                    disable.mutate({ id: schedule.id });
                                } else {
                                    enable.mutate({ id: schedule.id });
                                }
                            }}
                        >
                            {schedule.enabled ? copy.schedules.panel.disable : copy.schedules.panel.enable}
                        </Button>
                        <Button
                            variant="danger"
                            onClick={() => {
                                setDoomed(true);
                            }}
                        >
                            {copy.schedules.panel.delete}
                        </Button>
                    </>
                )}
            </div>

            {schedule === null ? null : (
                <>
                    <p className="text-2xs text-ink-faint">
                        {copy.schedules.panel.createdBy(actorLabel(schedule.createdBy))}
                    </p>
                    {schedule.lastRunId === null || schedule.lastRunId === undefined ? null : (
                        <LastRunCard siteId={siteId} runId={schedule.lastRunId} />
                    )}
                </>
            )}

            <Dialog
                open={doomed}
                onOpenChange={setDoomed}
                title={copy.schedules.panel.delete}
                description={copy.schedules.panel.deleteBody}
                confirmLabel={copy.schedules.panel.delete}
                cancelLabel={copy.app.cancel}
                destructive={true}
                busy={remove.isPending}
                onConfirm={() => {
                    if (schedule === null) {
                        return;
                    }
                    remove.mutate(
                        { id: schedule.id },
                        {
                            onSuccess: () => {
                                setDoomed(false);
                                onDeleted();
                            },
                        },
                    );
                }}
            />
        </div>
    );
}
