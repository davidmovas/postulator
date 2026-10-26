import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { EditNoteIcon, SectionLabel, toneClasses, WarningIcon, cx } from "../../ui/index.js";
import { fieldLabel } from "../settings/types/labels.js";

const said = copy.runs.start;

export interface TypeScopeProps {
    fields: readonly string[];
}

export function TypeScope({ fields }: TypeScopeProps): ReactElement {
    return (
        <p data-run-type-scope={true} className="flex items-start gap-2 text-xs text-ink-soft">
            <EditNoteIcon size={14} className="mt-0.5 shrink-0 text-ink-faint" />
            {said.typeScope(fields.map((field) => fieldLabel(field)))}
        </p>
    );
}

export interface ScopedAnswer {
    estimate: unknown;
    fields?: readonly string[] | null;
}

export function scopedFields(answered: ScopedAnswer): readonly string[] {
    return answered.fields ?? [];
}

export interface StartScopeProps {
    fields: readonly string[];
    rewritten: readonly string[];
}

export function StartScope({ fields, rewritten }: StartScopeProps): ReactElement {
    const labels = fields.map((field) => fieldLabel(field));
    return (
        <section aria-label={said.scopeTitle} data-run-scope={true} className="flex flex-col gap-1.5">
            <SectionLabel>{said.scopeTitle}</SectionLabel>
            <p className="flex items-start gap-2 text-xs text-ink-soft">
                <EditNoteIcon size={14} className="mt-0.5 shrink-0 text-ink-faint" />
                {said.fieldScope(labels)}
            </p>
            {rewritten.length === 0 ? null : (
                <p className={cx("flex items-start gap-2 text-xs", toneClasses.danger.ink)}>
                    <WarningIcon size={14} className="mt-0.5 shrink-0" />
                    {said.rewriteField(labels, rewritten)}
                </p>
            )}
        </section>
    );
}
