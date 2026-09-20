import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { failure } from "../../data/errors.js";
import { useTemplate } from "../../data/hooks/templates.js";
import type { Run } from "../../data/types.js";
import { absoluteTime, duration, relativeTime, tokens as formatTokens, usd as formatUsd } from "../../domain/format.js";
import type { IconComponent } from "../../ui/index.js";
import {
    AlarmIcon,
    BudgetGauge,
    DashboardCustomizeIcon,
    EditNoteIcon,
    ProgressBar,
    ScheduleIcon,
    SectionLabel,
} from "../../ui/index.js";
import type { RunView, StatsView } from "./authority.js";
import { stepLabel } from "./labels.js";
import { recipeSteps } from "./recipe.js";
import { spanMs } from "./span.js";

interface FactProps {
    icon: IconComponent;
    children: string;
    title?: string;
}

function Fact({ icon: Icon, children, title }: FactProps): ReactElement {
    return (
        <span title={title} className="inline-flex min-w-0 items-center gap-1 text-2xs text-ink-faint">
            <Icon size={13} className="shrink-0" />
            <span className="truncate">{children}</span>
        </span>
    );
}

function templateFact(run: Run, named: ReturnType<typeof useTemplate>): string {
    if (run.templateId === "") {
        return copy.runs.detail.noTemplate;
    }
    const held = named.data?.template;
    if (held !== undefined) {
        return copy.runs.detail.template(held.name, run.templateVersion);
    }
    if (named.isError && failure(named.error).code === "NOT_FOUND") {
        return copy.runs.detail.templateGone;
    }
    return copy.app.loading;
}

function Facts({ run }: { run: Run }): ReactElement {
    const steps = recipeSteps(run);
    const named = useTemplate(run.templateId === "" ? null : run.templateId);
    return (
        <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1 md:col-span-3">
            <Fact icon={ScheduleIcon} title={absoluteTime(run.startedAt)}>
                {`${copy.runs.detail.startedBy(run.createdBy)} · ${relativeTime(run.startedAt)}`}
            </Fact>
            <Fact icon={EditNoteIcon}>{copy.runs.detail.publishMode(run.publishMode)}</Fact>
            <Fact icon={AlarmIcon} title={absoluteTime(run.deadlineAt)}>
                {`${copy.runs.detail.deadline} ${relativeTime(run.deadlineAt)}`}
            </Fact>
            <Fact icon={DashboardCustomizeIcon} title={run.templateId}>
                {templateFact(run, named)}
            </Fact>
            {steps.length === 0 ? null : (
                <Fact icon={DashboardCustomizeIcon}>
                    {copy.runs.step.recipe(
                        stepLabel(steps[0]),
                        stepLabel(steps[steps.length - 1]),
                        steps.length,
                    )}
                </Fact>
            )}
        </div>
    );
}

export interface RunProgressProps {
    run: Run;
    view: RunView;
    stats: StatsView;
    terminal: boolean;
    now: number;
}

export function RunProgress({ run, view, stats, terminal, now }: RunProgressProps): ReactElement {
    const counted = terminal ? copy.runs.detail.rowStats : copy.runs.detail.liveStats;
    const pending = Math.max(stats.items - stats.done - stats.failed, 0);
    const tokenCap = run.budget.maxTokens;
    const lasted = spanMs(run.startedAt, run.finishedAt, now);

    return (
        <div className="grid shrink-0 grid-cols-1 gap-x-4 gap-y-3 border-b border-hairline px-3 py-3 md:grid-cols-3">
            <Facts run={run} />
            <div className="flex flex-col gap-1" title={counted}>
                <SectionLabel>{copy.runs.columns.progress}</SectionLabel>
                <ProgressBar
                    value={stats.done + stats.failed}
                    max={Math.max(stats.items, 1)}
                    tone={stats.failed > 0 ? "warn" : "accent"}
                    label={copy.runs.columns.progress}
                    leading={copy.runs.detail.items(stats.done, stats.items)}
                    trailing={lasted === null ? "" : duration(lasted)}
                />
                <div className="flex flex-wrap gap-x-3 font-mono text-2xs text-ink-faint">
                    <span>{`${String(stats.done)} ${copy.runs.detail.done}`}</span>
                    <span>{`${String(stats.failed)} ${copy.runs.detail.failed}`}</span>
                    <span>{`${String(pending)} ${copy.runs.detail.pending}`}</span>
                    {stats.needsHuman === null ? null : (
                        <span>{`${String(stats.needsHuman)} ${copy.runs.detail.needsHuman}`}</span>
                    )}
                </div>
            </div>

            <div className="flex flex-col gap-1" title={view.capped ? copy.runs.detail.hardStop : undefined}>
                <SectionLabel>{copy.runs.detail.spent}</SectionLabel>
                {view.capped ? (
                    <BudgetGauge
                        value={stats.usd}
                        max={view.cap}
                        hardStop={view.cap}
                        label={copy.runs.detail.spent}
                        leading={copy.runs.detail.spentOf(formatUsd(stats.usd), formatUsd(view.cap))}
                        trailing={stats.calls === null ? "" : copy.runs.detail.calls(stats.calls)}
                    />
                ) : (
                    <span className="font-mono text-sm text-ink">{formatUsd(stats.usd)}</span>
                )}
            </div>

            <div className="flex flex-col gap-1" title={copy.runs.detail.tokensBody}>
                <SectionLabel>{copy.runs.detail.tokens}</SectionLabel>
                {tokenCap > 0 ? (
                    <ProgressBar
                        value={stats.tokens}
                        max={tokenCap}
                        tone="info"
                        label={copy.runs.detail.tokens}
                        leading={copy.runs.detail.tokensOf(
                            formatTokens(stats.tokens),
                            formatTokens(tokenCap),
                        )}
                    />
                ) : (
                    <span className="font-mono text-sm text-ink">{formatTokens(stats.tokens)}</span>
                )}
            </div>
        </div>
    );
}
