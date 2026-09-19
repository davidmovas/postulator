import type { ReactElement } from "react";
import { Link } from "react-router";

import { copy } from "../../copy/index.js";
import {
    ArrowRightAltIcon,
    BlockIcon,
    Button,
    CheckCircleIcon,
    Panel,
    PanelHeader,
    ProgressBar,
    RadioButtonUncheckedIcon,
    SkeletonRows,
    StatusBadge,
    cx,
} from "../../ui/index.js";
import type { ReadinessStep } from "./readiness.js";
import { useReadiness } from "./readiness.js";

function StepMark({ step }: { step: ReadinessStep }): ReactElement {
    if (step.done) {
        return <CheckCircleIcon size={17} className="mt-px shrink-0 text-ok" />;
    }
    if (step.blocked) {
        return <BlockIcon size={17} className="mt-px shrink-0 text-ink-faint" />;
    }
    return <RadioButtonUncheckedIcon size={17} className="mt-px shrink-0 text-warn" />;
}

function StepRow({ step, index }: { step: ReadinessStep; index: number }): ReactElement {
    const body = step.done ? (step.hint ?? step.instruction) : step.instruction;
    const extra = step.done || step.hint === null || step.hint === step.instruction ? null : step.hint;

    return (
        <li className="flex items-start gap-2.5 border-b border-inset px-3 py-2 last:border-b-0">
            <StepMark step={step} />
            <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                <div className="flex min-w-0 items-center gap-2">
                    <span className="shrink-0 font-mono text-2xs text-ink-faint">{index + 1}</span>
                    <span
                        className={cx(
                            "truncate text-sm font-semibold",
                            step.done ? "text-ink-soft" : "text-ink",
                        )}
                    >
                        {step.name}
                    </span>
                </div>
                <p className="text-xs text-ink-dim">{body}</p>
                {extra === null ? null : <p className="text-xs text-warn">{extra}</p>}
            </div>
            {step.done ? null : (
                <Link
                    to={step.to}
                    className={cx(
                        "inline-flex h-6 shrink-0 items-center gap-1 rounded-sm px-2 text-xs font-semibold",
                        "text-accent transition-colors duration-100 ease-out hover:bg-accent-soft",
                    )}
                >
                    {step.action}
                    <ArrowRightAltIcon size={14} />
                </Link>
            )}
        </li>
    );
}

export interface ReadinessChecklistProps {
    siteId: string | null;
    className?: string;
}

export function ReadinessChecklist({ siteId, className }: ReadinessChecklistProps): ReactElement {
    const readiness = useReadiness(siteId);
    const left = readiness.total - readiness.done;

    return (
        <Panel className={className}>
            <PanelHeader title={copy.onboarding.panel}>
                <div className="flex shrink-0 items-center gap-2">
                    {readiness.ready ? (
                        <StatusBadge tone="ok">{copy.onboarding.ready}</StatusBadge>
                    ) : (
                        <StatusBadge tone="warn">{copy.onboarding.remaining(left)}</StatusBadge>
                    )}
                    <Button
                        size="sm"
                        variant="ghost"
                        onClick={readiness.refresh}
                        busy={readiness.loading}
                    >
                        {copy.onboarding.recheck}
                    </Button>
                </div>
            </PanelHeader>
            <div className="border-b border-hairline px-3 py-2">
                <ProgressBar
                    label={copy.onboarding.panel}
                    value={readiness.done}
                    max={readiness.total}
                    tone={readiness.ready ? "ok" : "accent"}
                    leading={
                        readiness.siteName === null
                            ? copy.onboarding.noSite
                            : copy.onboarding.forSite(readiness.siteName)
                    }
                    trailing={copy.onboarding.progress(readiness.done, readiness.total)}
                />
            </div>
            {readiness.loading && readiness.done === 0 ? (
                <SkeletonRows rows={8} height={22} label={copy.app.loading} className="p-3" />
            ) : (
                <ul className="flex flex-col">
                    {readiness.steps.map((step, index) => (
                        <StepRow key={step.id} step={step} index={index} />
                    ))}
                </ul>
            )}
        </Panel>
    );
}
