import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import { useSyncProgress } from "../../../data/hooks/sync.js";
import type { SyncProgress } from "../../../data/types.js";
import { Banner, CheckSmallIcon, cx, SectionLabel, SkeletonRows, Spinner, useNow } from "../../../ui/index.js";
import type { StageState, StageStep } from "../../sync/index.js";
import { leftReasons, pageCount, stageSteps, stageWords, SyncError, syncRetrying } from "../../sync/index.js";
import { countdown, remainingMs } from "../authority.js";
import { statusCancelled, statusCompleted, statusFailed, statusPaused } from "../statuses.js";

const said = copy.sync;

const tickMs = 1000;

const markerTone: Readonly<Record<StageState, string>> = {
    done: "border-hairline bg-panel text-ink-dim",
    current: "border-accent bg-accent-soft text-accent",
    failed: "border-danger bg-danger-soft text-danger",
    stopped: "border-muted-border bg-muted-soft text-muted",
    pending: "border-hairline bg-panel text-ink-faint",
};

const wordTone: Readonly<Record<StageState, string>> = {
    done: "text-ink-soft",
    current: "font-semibold text-ink",
    failed: "font-semibold text-danger",
    stopped: "font-semibold text-ink-soft",
    pending: "text-ink-faint",
};

function Marker({ state, moving }: { state: StageState; moving: boolean }): ReactElement {
    return (
        <span
            aria-hidden={true}
            className={cx(
                "relative z-10 flex h-3.5 w-3.5 shrink-0 items-center justify-center rounded-full border",
                markerTone[state],
            )}
        >
            {state === "done" ? <CheckSmallIcon size={12} /> : null}
            {state === "current" && moving ? <Spinner size={7} /> : null}
            {state === "current" && !moving ? <span className="h-1 w-1 rounded-full bg-current" /> : null}
            {state === "failed" || state === "stopped" ? <span className="h-1 w-1 rounded-full bg-current" /> : null}
        </span>
    );
}

function StageRail({ steps, moving }: { steps: readonly StageStep[]; moving: boolean }): ReactElement {
    return (
        <ol
            aria-label={said.stagesLabel}
            className="relative flex min-w-0 flex-col before:absolute before:top-2.5 before:bottom-2.5 before:left-[6.5px] before:w-px before:bg-hairline"
        >
            {steps.map((step) => (
                <li
                    key={step.stage}
                    data-stage-state={step.state}
                    aria-current={step.state === "current" ? "step" : undefined}
                    className="flex min-w-0 items-center gap-2.5 py-1"
                >
                    <Marker state={step.state} moving={moving} />
                    <span className={cx("truncate text-xs", wordTone[step.state])}>{step.words}</span>
                </li>
            ))}
        </ol>
    );
}

function Counts({ progress }: { progress: SyncProgress }): ReactElement {
    const entries: readonly (readonly [string, number, boolean])[] = [
        [said.counts.pulled, progress.pulled, false],
        [said.counts.created, progress.created, false],
        [said.counts.updated, progress.updated, false],
        [said.counts.products, progress.products, false],
        [said.counts.rendered, progress.rendered, false],
        [said.counts.unreachable, progress.unreachable, progress.unreachable > 0],
        [said.counts.archived, progress.archived, false],
        [said.counts.entities, progress.entities, false],
        [said.counts.placed, progress.placed, false],
        [said.counts.unplaced, progress.unplaced, progress.unplaced > 0],
        [said.counts.batches, progress.batches, false],
    ];
    return (
        <section aria-label={said.counts.title} className="flex min-w-0 flex-col gap-1.5">
            <SectionLabel>{said.counts.title}</SectionLabel>
            <dl className="grid grid-cols-[minmax(0,1fr)_auto] gap-x-4 gap-y-1 text-xs @md:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)_auto]">
                {entries.map(([label, value, flagged]) => (
                    <div key={label} className="contents">
                        <dt className="truncate text-ink-dim">{label}</dt>
                        <dd className={cx("text-right font-mono tabular-nums", flagged ? "text-warn" : "text-ink")}>
                            {pageCount(value)}
                        </dd>
                    </div>
                ))}
            </dl>
        </section>
    );
}

function Left({ progress }: { progress: SyncProgress }): ReactElement | null {
    const reasons = leftReasons(progress);
    if (reasons.length === 0) {
        return null;
    }
    return (
        <section className="flex min-w-0 flex-col gap-1.5">
            <SectionLabel>{said.left.title}</SectionLabel>
            <ul aria-label={said.left.title} className="flex min-w-0 flex-col gap-1 text-xs">
                {reasons.map((left) => (
                    <li key={left.reason} className="flex min-w-0 items-baseline justify-between gap-4">
                        <span className="min-w-0 text-ink-dim">{left.words}</span>
                        <span className="shrink-0 text-right font-mono tabular-nums text-ink">{pageCount(left.count)}</span>
                    </li>
                ))}
            </ul>
        </section>
    );
}

function Body({ progress }: { progress: SyncProgress }): ReactElement {
    const retrying = syncRetrying(progress);
    const now = useNow(tickMs, retrying);
    const left = retrying ? remainingMs(progress.wakeAt, now) : null;
    const stopped = progress.status === statusFailed || progress.status === statusCancelled;
    const moving = !stopped && progress.status !== statusPaused && progress.status !== statusCompleted;

    return (
        <section aria-label={said.pane} data-sync-pane={progress.runId} className="@container flex min-w-0 flex-col gap-3 p-3">
            <div className="flex min-w-0 flex-wrap items-baseline justify-between gap-x-3 gap-y-1">
                <h3
                    className={cx(
                        "min-w-0 text-base font-semibold",
                        progress.status === statusFailed ? "text-danger" : "text-ink",
                    )}
                >
                    {stageWords(progress.stage)}
                </h3>
                <span
                    className="shrink-0 text-xs text-ink-dim"
                    title={progress.full ? said.walk.fullBody : said.walk.incrementalBody}
                >
                    {progress.full ? said.walk.full : said.walk.incremental}
                </span>
            </div>
            <div className="grid min-w-0 gap-4 @lg:grid-cols-[minmax(11rem,15rem)_minmax(0,1fr)]">
                <StageRail steps={stageSteps(progress)} moving={moving} />
                <div className="flex min-w-0 flex-col gap-3">
                    <Counts progress={progress} />
                    <Left progress={progress} />
                    {left === null ? null : (
                        <p className="text-xs text-warn">{`${said.attempt(progress.attempts)}; ${said.retryIn(countdown(left))}`}</p>
                    )}
                    {progress.error === "" ? null : (
                        <section aria-label={stopped ? said.error : said.lastError} className="flex min-w-0 flex-col gap-1.5">
                            <SectionLabel>{stopped ? said.error : said.lastError}</SectionLabel>
                            <SyncError error={progress.error} />
                        </section>
                    )}
                </div>
            </div>
        </section>
    );
}

export interface SiteSyncPaneProps {
    siteId: string;
    runId: string;
}

export function SiteSyncPane({ siteId, runId }: SiteSyncPaneProps): ReactElement {
    const progress = useSyncProgress(siteId, runId);

    if (progress.isPending) {
        return (
            <div className="p-3">
                <SkeletonRows rows={6} label={said.loading} />
            </div>
        );
    }

    if (progress.data === undefined || !progress.data.found) {
        return (
            <div className="p-3">
                <Banner tone="warn" title={said.gone} />
            </div>
        );
    }

    return <Body progress={progress.data} />;
}
