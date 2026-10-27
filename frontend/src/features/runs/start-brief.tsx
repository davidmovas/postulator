import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../copy/index.js";
import { ChevronRightIcon, cx, Field, Input, Textarea } from "../../ui/index.js";
import type { BriefDraft, BriefKey, BriefLine, BriefText } from "./start-directions.js";
import { briefLines, briefProblems, briefTexts, filledOf, limitOf, runes } from "./start-directions.js";

const said = copy.runs.start.brief;

const labels: Readonly<Record<BriefKey, { label: string; hint: string }>> = {
    goal: { label: said.goal, hint: said.goalHint },
    focus: { label: said.focus, hint: said.focusHint },
    tone: { label: said.tone, hint: said.toneHint },
    avoid: { label: said.avoid, hint: said.avoidHint },
    reference: { label: said.reference, hint: said.referenceHint },
    styleSample: { label: said.styleSample, hint: said.styleSampleHint },
};

const textRows: Readonly<Record<BriefText, number>> = { reference: 6, styleSample: 4 };

const countFrom = 0.8;

function noteOf(key: BriefKey, value: string): string {
    const used = runes(value);
    const limit = limitOf(key);
    return used >= limit * countFrom ? said.count(used, limit) : labels[key].hint;
}

export interface BriefSectionProps {
    draft: BriefDraft;
    onChange: (next: BriefDraft) => void;
}

export function BriefSection({ draft, onChange }: BriefSectionProps): ReactElement {
    const [open, setOpen] = useState(false);
    const problems = briefProblems(draft);
    const filled = filledOf(draft);
    const all = briefLines.length + briefTexts.length;

    const set = (key: BriefKey, value: string): void => {
        onChange({ ...draft, [key]: value });
    };

    const line = (key: BriefLine): ReactElement => (
        <Field
            key={key}
            label={labels[key].label}
            hint={noteOf(key, draft[key])}
            error={problems.has(key) ? said.tooLong(limitOf(key)) : null}
        >
            {(control) => (
                <Input
                    id={control.id}
                    aria-describedby={control["aria-describedby"]}
                    invalid={control.invalid}
                    data-run-brief={key}
                    value={draft[key]}
                    onChange={(event) => {
                        set(key, event.target.value);
                    }}
                />
            )}
        </Field>
    );

    const text = (key: BriefText): ReactElement => (
        <Field
            key={key}
            label={labels[key].label}
            hint={noteOf(key, draft[key])}
            error={problems.has(key) ? said.tooLong(limitOf(key)) : null}
        >
            {(control) => (
                <Textarea
                    id={control.id}
                    aria-describedby={control["aria-describedby"]}
                    invalid={control.invalid}
                    data-run-brief={key}
                    rows={textRows[key]}
                    value={draft[key]}
                    onChange={(event) => {
                        set(key, event.target.value);
                    }}
                />
            )}
        </Field>
    );

    return (
        <section
            aria-label={said.title}
            data-run-brief-section={open ? "open" : "closed"}
            className="flex flex-col rounded-md border border-hairline bg-panel"
        >
            <button
                type="button"
                aria-expanded={open}
                title={open ? said.close : said.open}
                onClick={() => {
                    setOpen((held) => !held);
                }}
                className="flex min-w-0 items-center gap-2 px-3 py-2 text-left hover:bg-inset"
            >
                <ChevronRightIcon
                    size={14}
                    className={cx("shrink-0 text-ink-dim transition-transform duration-100", open && "rotate-90")}
                />
                <span className="flex min-w-0 flex-1 flex-col">
                    <span className="text-xs font-semibold text-ink">{said.title}</span>
                    <span className="truncate text-2xs text-ink-faint">{said.hint}</span>
                </span>
                <span
                    className={cx(
                        "shrink-0 font-mono text-2xs",
                        problems.size > 0 ? "text-danger" : filled === 0 ? "text-ink-faint" : "text-accent",
                    )}
                >
                    {said.filled(filled, all)}
                </span>
            </button>
            {open ? (
                <div className="flex flex-col gap-2.5 border-t border-hairline px-3 py-2.5">
                    <div className="grid grid-cols-2 gap-2">{briefLines.map(line)}</div>
                    {briefTexts.map(text)}
                </div>
            ) : null}
        </section>
    );
}
