import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { Run } from "../../data/types.js";
import { duration, tokens as formatTokens, usd as formatUsd } from "../../domain/format.js";
import { BudgetGauge, ProgressBar, SectionLabel } from "../../ui/index.js";
import type { RunView, StatsView } from "./authority.js";
import { spanMs } from "./span.js";

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
        <div className="grid shrink-0 grid-cols-1 gap-4 border-b border-hairline px-3 py-3 md:grid-cols-3">
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
                    <div className="flex flex-col gap-0.5">
                        <span className="font-mono text-sm text-ink">{formatUsd(stats.usd)}</span>
                        <span className="font-mono text-2xs text-warn">{copy.runs.detail.noCap}</span>
                    </div>
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
