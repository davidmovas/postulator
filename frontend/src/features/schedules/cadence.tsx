import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { cadenceWords } from "../../domain/cron.js";
import { Field, Input, Segmented, Select } from "../../ui/index.js";
import type { CadenceKind, IntervalUnit, ScheduleDraft } from "./form.js";
import { cadenceOf, intervalUnits } from "./form.js";

const unitOptions = intervalUnits.map((unit) => ({ value: unit, label: copy.schedules.panel.units[unit] }));

export interface CadenceFieldsProps {
    draft: ScheduleDraft;
    cronError: string | null;
    onChange: (next: ScheduleDraft) => void;
}

export function CadenceFields({ draft, cronError, onChange }: CadenceFieldsProps): ReactElement {
    const cadence = cadenceOf(draft);
    const words = cadenceWords(cadence.cron, cadence.intervalMinutes);

    return (
        <div className="flex flex-col gap-2">
            <Segmented<CadenceKind>
                label={copy.schedules.panel.cadence}
                value={draft.cadence}
                options={[
                    { value: "interval", label: copy.schedules.panel.interval },
                    { value: "cron", label: copy.schedules.panel.cron },
                ]}
                onValueChange={(next) => {
                    onChange({ ...draft, cadence: next });
                }}
            />
            {draft.cadence === "cron" ? (
                <Field label={copy.schedules.panel.cron} error={cronError}>
                    {(control) => (
                        <Input
                            id={control.id}
                            data-schedule-cron={true}
                            mono={true}
                            invalid={control.invalid}
                            value={draft.cron}
                            placeholder={copy.schedules.panel.cronPlaceholder}
                            onChange={(event) => {
                                onChange({ ...draft, cron: event.target.value });
                            }}
                        />
                    )}
                </Field>
            ) : (
                <Field label={copy.schedules.panel.every} error={cronError}>
                    {(control) => (
                        <div className="flex items-center gap-2">
                            <Input
                                id={control.id}
                                type="number"
                                min={1}
                                mono={true}
                                invalid={control.invalid}
                                value={String(draft.intervalValue)}
                                className="w-20"
                                onChange={(event) => {
                                    onChange({ ...draft, intervalValue: Number(event.target.value) });
                                }}
                            />
                            <Select<IntervalUnit>
                                aria-label={copy.schedules.panel.cadence}
                                value={draft.intervalUnit}
                                options={unitOptions}
                                onValueChange={(unit) => {
                                    onChange({ ...draft, intervalUnit: unit });
                                }}
                            />
                        </div>
                    )}
                </Field>
            )}
            <p className="text-2xs text-ink-dim">{words === "" ? copy.schedules.cadence.none : words}</p>
        </div>
    );
}
