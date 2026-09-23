import type { ReactElement } from "react";
import { Link } from "react-router";

import { copy } from "../../copy/index.js";
import {
    ArrowRightAltIcon,
    BlockIcon,
    Button,
    CheckCircleIcon,
    cx,
    Panel,
    PanelHeader,
    ProgressBar,
    RadioButtonUncheckedIcon,
    SkeletonRows,
    StatusBadge,
} from "../../ui/index.js";
import type { Readiness, ReadinessStep } from "./readiness.js";

function StepMark({ step }: { step: ReadinessStep }): ReactElement {
    if (step.done) {
        return <CheckCircleIcon size={15} className="mt-px shrink-0 text-ok" />;
    }
    if (step.blocked) {
        return <BlockIcon size={15} className="mt-px shrink-0 text-ink-faint" />;
    }
    return <RadioButtonUncheckedIcon size={15} className="mt-px shrink-0 text-warn" />;
}

function StepRow({ step }: { step: ReadinessStep }): ReactElement {
    const body = step.done ? (step.hint ?? step.instruction) : step.instruction;

    return (
        <li className="flex items-start gap-2 border-b border-inset px-3 py-1.5 last:border-b-0">
            <StepMark step={step} />
            <div className="flex min-w-0 flex-1 flex-col">
                <span className={cx("truncate text-xs font-semibold", step.done ? "text-ink-soft" : "text-ink")}>
                    {step.name}
                </span>
                <span className="text-2xs text-ink-dim">{body}</span>
            </div>
            {step.done ? null : (
                <Link
                    to={step.to}
                    title={step.action}
                    className={cx(
                        "inline-flex h-5 shrink-0 items-center gap-0.5 rounded-sm px-1 text-2xs font-semibold",
                        "text-accent transition-colors duration-100 ease-out hover:bg-accent-soft",
                    )}
                >
                    {step.action}
                    <ArrowRightAltIcon size={13} />
                </Link>
            )}
        </li>
    );
}

export interface ReadinessChecklistProps {
    readiness: Readiness;
}

export function ReadinessChecklist({ readiness }: ReadinessChecklistProps): ReactElement {
    const left = readiness.total - readiness.done;

    return (
        <Panel>
            <PanelHeader title={copy.readiness.panel}>
                <div className="flex shrink-0 items-center gap-2">
                    {readiness.ready ? (
                        <StatusBadge tone="ok">{copy.readiness.ready}</StatusBadge>
                    ) : (
                        <StatusBadge tone="warn">{copy.readiness.remaining(left)}</StatusBadge>
                    )}
                    <Button size="sm" variant="ghost" onClick={readiness.refresh} busy={readiness.loading}>
                        {copy.readiness.recheck}
                    </Button>
                </div>
            </PanelHeader>
            <div className="border-b border-hairline px-3 py-2">
                <ProgressBar
                    label={copy.readiness.panel}
                    value={readiness.done}
                    max={readiness.total}
                    tone={readiness.ready ? "ok" : "accent"}
                    trailing={copy.readiness.progress(readiness.done, readiness.total)}
                />
            </div>
            {readiness.loading && readiness.done === 0 ? (
                <SkeletonRows rows={8} height={20} label={copy.app.loading} className="p-3" />
            ) : (
                <ul className="flex flex-col">
                    {readiness.steps.map((step) => (
                        <StepRow key={step.id} step={step} />
                    ))}
                </ul>
            )}
        </Panel>
    );
}
