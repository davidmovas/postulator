import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { react } from "../../data/errors.js";
import { useRunSpend } from "../../data/hooks/models.js";
import { usd } from "../../domain/format.js";
import type { StepRow } from "../../domain/spend.js";
import { percent, stepRows } from "../../domain/spend.js";
import { Banner, Panel, PanelHeader, SkeletonRows } from "../../ui/index.js";
import { stepLabel } from "../runs/labels.js";

const said = copy.reports.runs.costByStep;

function StepCost({ row }: { row: StepRow }): ReactElement {
    return (
        <li className="flex flex-col gap-1 border-b border-hairline p-3 last:border-b-0">
            <div className="flex items-center gap-2">
                <h3 className="min-w-0 flex-1 truncate text-sm text-ink">
                    {row.step === "" ? said.unnamed : stepLabel(row.step)}
                </h3>
                <span className="shrink-0 font-mono text-xs text-ink">{usd(row.usd)}</span>
            </div>
            <div className="flex items-center gap-2">
                <span aria-hidden={true} className="h-1 w-16 shrink-0 overflow-hidden rounded-sm bg-inset">
                    <span className="block h-full bg-accent" style={{ width: `${Math.min(100, row.share * 100)}%` }} />
                </span>
                <span className="font-mono text-2xs text-ink-dim">{said.ofRun(percent(row.share))}</span>
            </div>
            <div className="flex flex-wrap items-center gap-x-2 text-2xs text-ink-dim">
                <span>{said.calls(row.calls)}</span>
                {row.output > 0 ? (
                    <span title={said.reasoningHint}>{said.reasoning(percent(row.reasoningShare))}</span>
                ) : null}
                {row.failed > 0 ? <span className="text-danger">{said.failed(row.failed)}</span> : null}
            </div>
        </li>
    );
}

export function RunCostPanel({ runId }: { runId: string }): ReactElement {
    const spend = useRunSpend(runId === "" ? null : runId);
    const failure = spend.error === null ? null : react(spend.error);
    const shownFailure =
        spend.data === undefined && failure !== null && failure.kind !== "silent" && failure.kind !== "unlock"
            ? failure.message
            : null;
    const rows = stepRows(spend.data?.slices ?? null);

    return (
        <Panel>
            <PanelHeader title={said.title} />
            {shownFailure !== null ? (
                <div className="p-3">
                    <Banner tone="danger" title={shownFailure} />
                </div>
            ) : spend.data === undefined ? (
                <div className="p-3">
                    <SkeletonRows rows={3} label={said.loading} />
                </div>
            ) : rows.length === 0 ? (
                <p className="p-3 text-xs text-ink-dim">{said.none}</p>
            ) : (
                <ul aria-label={said.title} className="flex flex-col">
                    {rows.map((row) => (
                        <StepCost key={row.step} row={row} />
                    ))}
                </ul>
            )}
        </Panel>
    );
}
