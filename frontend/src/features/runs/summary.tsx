import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { RunEventsState } from "../../data/runs/log.js";
import type { Run } from "../../data/types.js";
import { absoluteTime, relativeTime, tokens as formatTokens, usd as formatUsd } from "../../domain/format.js";
import type { IconComponent } from "../../ui/index.js";
import {
    AlarmIcon,
    Banner,
    BudgetGauge,
    CloudSyncIcon,
    DashboardCustomizeIcon,
    EditNoteIcon,
    ProgressBar,
    ScheduleIcon,
    SectionLabel,
    StatusBadge,
} from "../../ui/index.js";
import type { StatsView } from "./authority.js";
import { runView } from "./authority.js";
import { RunControls } from "./controls.js";
import { pauseReasonText, pauseReasonTone, statusIcon, statusLabel, statusTone } from "./labels.js";
import { statusFailed } from "./statuses.js";

interface ChipProps {
    icon: IconComponent;
    children: string;
    title?: string;
}

function Chip({ icon: Icon, children, title }: ChipProps): ReactElement {
    return (
        <span title={title} className="inline-flex min-w-0 items-center gap-1 text-2xs text-ink-faint">
            <Icon size={13} className="shrink-0" />
            <span className="truncate">{children}</span>
        </span>
    );
}

export interface RunSummaryProps {
    run: Run;
    stats: StatsView;
    events: RunEventsState;
    gap: boolean;
    terminal: boolean;
    steps: readonly string[];
}

export function RunSummary({ run, stats, events, gap, terminal, steps }: RunSummaryProps): ReactElement {
    const view = runView(run);
    const pending = Math.max(stats.items - stats.done - stats.failed, 0);

    return (
        <div className="flex shrink-0 flex-col gap-3 border-b border-hairline px-3 py-3">
            <div className="flex items-start justify-between gap-3">
                <div className="flex min-w-0 flex-col gap-1.5">
                    <div className="flex min-w-0 flex-wrap items-center gap-2">
                        <h1 className="truncate text-sm font-semibold text-ink">
                            {copy.runs.detail.header(run.kind)}
                        </h1>
                        <StatusBadge tone={statusTone(run.status)} icon={statusIcon(run.status)}>
                            {statusLabel(run.status)}
                        </StatusBadge>
                        {view.paused && run.pauseReason !== "" ? (
                            <StatusBadge tone={pauseReasonTone(run.pauseReason)}>
                                {pauseReasonText(run.pauseReason)}
                            </StatusBadge>
                        ) : null}
                        <span className="truncate font-mono text-2xs text-ink-faint">{run.id}</span>
                    </div>
                    <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1">
                        <Chip icon={ScheduleIcon} title={absoluteTime(run.startedAt)}>
                            {`${copy.runs.detail.startedBy(run.createdBy)} · ${relativeTime(run.startedAt)}`}
                        </Chip>
                        <Chip icon={EditNoteIcon}>{copy.runs.detail.publishMode(run.publishMode)}</Chip>
                        <Chip icon={AlarmIcon} title={absoluteTime(run.deadlineAt)}>
                            {`${copy.runs.detail.deadline} ${relativeTime(run.deadlineAt)}`}
                        </Chip>
                        <Chip icon={DashboardCustomizeIcon}>
                            {run.templateId === ""
                                ? copy.runs.detail.noTemplate
                                : copy.runs.detail.template(run.templateId, run.templateVersion)}
                        </Chip>
                        {steps.length === 0 ? null : (
                            <Chip icon={DashboardCustomizeIcon}>
                                {copy.runs.step.recipe(steps[0], steps[steps.length - 1], steps.length)}
                            </Chip>
                        )}
                    </div>
                </div>
                <RunControls run={run} />
            </div>

            <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
                <div className="flex flex-col gap-1">
                    <SectionLabel>{copy.runs.columns.progress}</SectionLabel>
                    <ProgressBar
                        value={stats.done + stats.failed}
                        max={Math.max(stats.items, 1)}
                        tone={stats.failed > 0 ? "warn" : "accent"}
                        label={copy.runs.columns.progress}
                        leading={copy.runs.detail.items(stats.done, stats.items)}
                        trailing={`${pending} ${copy.runs.detail.pending}`}
                    />
                    <div className="flex flex-wrap gap-x-3 font-mono text-2xs text-ink-faint">
                        <span>{`${stats.done} ${copy.runs.detail.done}`}</span>
                        <span>{`${stats.failed} ${copy.runs.detail.failed}`}</span>
                        {stats.needsHuman === null ? null : (
                            <span>{`${stats.needsHuman} ${copy.runs.detail.needsHuman}`}</span>
                        )}
                    </div>
                    <p className="text-2xs text-ink-faint">
                        {terminal ? copy.runs.detail.rowStats : copy.runs.detail.liveStats}
                    </p>
                </div>

                <div className="flex flex-col gap-1">
                    <SectionLabel>{copy.runs.detail.spent}</SectionLabel>
                    {view.capped ? (
                        <BudgetGauge
                            value={stats.usd}
                            max={view.cap}
                            hardStop={view.cap}
                            label={copy.runs.detail.spent}
                            leading={copy.runs.detail.spentOf(formatUsd(stats.usd), formatUsd(view.cap))}
                            trailing={stats.calls === null ? "" : copy.runs.detail.calls(stats.calls)}
                            footLeading={copy.runs.detail.hardStop}
                        />
                    ) : (
                        <div className="flex flex-col gap-0.5">
                            <span className="font-mono text-sm text-ink">{formatUsd(stats.usd)}</span>
                            <span className="text-2xs text-warn">{copy.runs.detail.noCap}</span>
                        </div>
                    )}
                </div>

                <div className="flex flex-col gap-1">
                    <SectionLabel>{copy.runs.detail.tokens}</SectionLabel>
                    <span className="font-mono text-sm text-ink">{formatTokens(stats.tokens)}</span>
                    <span className="text-2xs text-ink-faint">{copy.runs.detail.tokensBody}</span>
                </div>
            </div>

            {gap ? (
                <Banner
                    tone="info"
                    icon={CloudSyncIcon}
                    title={copy.runs.reconnecting}
                    body={copy.runs.reconnectingBody}
                />
            ) : null}
            {events.phase === "error" && events.error !== null ? (
                <Banner tone="warn" title={copy.runs.logFailed} body={events.error.message} />
            ) : null}
            {view.budgetPaused ? (
                <Banner
                    tone="danger"
                    title={copy.runs.budgetPausedTitle}
                    body={copy.runs.budgetPausedBody}
                />
            ) : view.paused ? (
                <Banner tone={pauseReasonTone(run.pauseReason)} title={pauseReasonText(run.pauseReason)} />
            ) : null}
            {view.status === statusFailed && run.error !== "" ? (
                <Banner tone="danger" title={copy.runs.detail.failure} body={run.error} />
            ) : null}
            {!view.capped && !view.terminal ? (
                <Banner tone="warn" title={copy.runs.detail.noCap} body={copy.runs.detail.noCapBody} />
            ) : null}
        </div>
    );
}
