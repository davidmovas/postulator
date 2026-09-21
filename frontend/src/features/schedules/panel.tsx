import type { ReactElement } from "react";
import { useEffect, useState } from "react";

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
import { Banner, Button, CloseIcon, Dialog, Field, IconButton, Input, PlayArrowIcon } from "../../ui/index.js";
import { ScheduleFields } from "./fields.js";
import { createOf, draftOf, dirty, ready, updateOf } from "./form.js";
import type { ScheduleDraft } from "./form.js";
import { actorLabel } from "./labels.js";
import { LastRunCard } from "./last-run.js";

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
    const formError = formErrorOf(thrown);
    const changed = dirty(draft, schedule);
    const saving = create.isPending || update.isPending;

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
            <header className="flex h-8 shrink-0 items-center justify-between gap-2 border-b border-hairline">
                <h2 className="min-w-0 truncate text-sm font-semibold text-ink">
                    {schedule?.name ?? copy.schedules.panel.create}
                </h2>
                <IconButton
                    icon={CloseIcon}
                    label={copy.schedules.panel.close}
                    size="sm"
                    variant="ghost"
                    onClick={onClose}
                />
            </header>
            <Field label={copy.schedules.panel.name} error={fieldErrorOf(thrown, "name")} required={true}>
                {(control) => (
                    <Input
                        id={control.id}
                        data-schedule-name={true}
                        value={draft.name}
                        invalid={control.invalid}
                        placeholder={copy.schedules.panel.namePlaceholder}
                        onChange={(event) => {
                            setDraft({ ...draft, name: event.target.value });
                        }}
                    />
                )}
            </Field>
            <ScheduleFields
                draft={draft}
                schedule={schedule}
                templates={templates}
                entities={entities}
                thrown={thrown}
                onChange={setDraft}
            />

            {formError === null ? null : <Banner tone="danger" title={formError} />}
            {runNow.data?.skipped === undefined || runNow.data.skipped === "" ? null : (
                <Banner tone="warn" title={runNow.data.skipped} />
            )}

            <div className="flex flex-wrap gap-2">
                <Button
                    variant="primary"
                    data-schedule-save={true}
                    busy={saving}
                    disabled={!ready(draft) || !changed}
                    onClick={save}
                >
                    {copy.schedules.panel.save}
                </Button>
                {schedule === null ? null : (
                    <>
                        <Button
                            variant="secondary"
                            data-schedule-run={true}
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
                            data-schedule-delete={true}
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
