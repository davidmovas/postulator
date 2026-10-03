import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../../copy/index.js";
import { react } from "../../../data/errors.js";
import { useSpendReport } from "../../../data/hooks/models.js";
import type { SpendTotals } from "../../../data/types.js";
import { tokens, usd } from "../../../domain/format.js";
import type { ModelRow, PurposeRow } from "../../../domain/spend.js";
import { count, modelRows, percent, purposeRows } from "../../../domain/spend.js";
import type { SegmentedOption } from "../../../ui/index.js";
import {
    Banner,
    Button,
    DenseTable,
    EmptyState,
    MonitoringIcon,
    Panel,
    PanelHeader,
    Segmented,
    SkeletonRows,
    TableCell,
    TableHead,
    TableRow,
    cx,
} from "../../../ui/index.js";
import type { SpendDays } from "./spend-labels.js";
import { daysOf, defaultSpendDays, purposeLabel, spendDays, tierLabel } from "./spend-labels.js";

const said = copy.settings.models.spend;

const rangeOptions: readonly SegmentedOption<string>[] = spendDays.map((days) => ({
    value: String(days),
    label: said.days(days),
}));

const purposeColumns = "minmax(8rem,1.4fr) 4.5rem 4.5rem 5.5rem minmax(7rem,1fr)";

const modelColumns = "minmax(9rem,1.6fr) 5rem 4rem 4.5rem 4.5rem 4.5rem 4.5rem 5rem";

interface ShareTileProps {
    label: string;
    fraction: number;
    note: string;
}

function ShareTile({ label, fraction, note }: ShareTileProps): ReactElement {
    return (
        <div
            role="group"
            aria-label={label}
            className="flex min-w-0 flex-col gap-0.5 rounded-md border border-hairline bg-inset px-3 py-2"
        >
            <span className="text-2xs font-semibold tracking-label text-ink-faint uppercase">{label}</span>
            <span className="font-mono text-xl leading-none text-ink">{percent(fraction)}</span>
            <span className="text-2xs text-ink-dim">{note}</span>
        </div>
    );
}

interface TotalsProps {
    totals: SpendTotals;
    days: number;
    busy: boolean;
}

function Totals({ totals, days, busy }: TotalsProps): ReactElement {
    return (
        <div
            aria-busy={busy}
            className={cx(
                "grid grid-cols-1 gap-2 p-3 transition-opacity @xl:grid-cols-2 @3xl:grid-cols-4",
                busy && "opacity-60",
            )}
        >
            <div className="flex min-w-0 flex-col justify-center gap-1 px-1">
                <span className="font-mono text-3xl leading-none font-semibold tracking-tight text-ink">
                    {usd(totals.usd)}
                </span>
                <span className="text-xs text-ink-dim">{said.over(days)}</span>
                <span className="flex flex-wrap gap-x-2 font-mono text-xs">
                    <span className="text-ink-soft">{said.calls(totals.calls)}</span>
                    {totals.failed > 0 ? <span className="text-danger">{said.failed(totals.failed)}</span> : null}
                </span>
            </div>
            <ShareTile label={said.shares.cached.label} fraction={totals.cachedShare} note={said.shares.cached.note} />
            <ShareTile
                label={said.shares.reasoning.label}
                fraction={totals.reasoningShare}
                note={said.shares.reasoning.note}
            />
            <ShareTile label={said.shares.flex.label} fraction={totals.flexShare} note={said.shares.flex.note} />
        </div>
    );
}

function ShareBar({ fraction }: { fraction: number }): ReactElement {
    return (
        <span className="flex min-w-0 items-center gap-2">
            <span aria-hidden={true} className="h-1 min-w-0 flex-1 overflow-hidden rounded-sm bg-inset">
                <span className="block h-full bg-accent" style={{ width: `${Math.min(100, fraction * 100)}%` }} />
            </span>
            <span className="w-9 shrink-0 text-right font-mono text-xs text-ink-soft">{percent(fraction)}</span>
        </span>
    );
}

function PurposeTable({ rows }: { rows: readonly PurposeRow[] }): ReactElement {
    const named = said.byPurpose;
    return (
        <Panel>
            <PanelHeader title={named.title} />
            <DenseTable columns={purposeColumns} label={named.title}>
                <TableHead>
                    <span>{named.purpose}</span>
                    <span className="text-right">{named.calls}</span>
                    <span className="text-right">{named.failed}</span>
                    <span className="text-right">{named.spent}</span>
                    <span className="text-right">{named.share}</span>
                </TableHead>
                {rows.map((row) => (
                    <TableRow key={row.purpose}>
                        <TableCell>{purposeLabel(row.purpose)}</TableCell>
                        <TableCell mono={true} align="right">
                            {count(row.calls)}
                        </TableCell>
                        <TableCell mono={true} align="right" muted={row.failed === 0}>
                            <span className={row.failed > 0 ? "text-danger" : undefined}>{count(row.failed)}</span>
                        </TableCell>
                        <TableCell mono={true} align="right">
                            {usd(row.usd)}
                        </TableCell>
                        <TableCell>
                            <ShareBar fraction={row.share} />
                        </TableCell>
                    </TableRow>
                ))}
            </DenseTable>
        </Panel>
    );
}

function ModelTable({ rows }: { rows: readonly ModelRow[] }): ReactElement {
    const named = said.byModel;
    return (
        <Panel>
            <PanelHeader title={named.title} />
            <div className="overflow-x-auto">
                <DenseTable columns={modelColumns} label={named.title} className="min-w-[44rem]">
                    <TableHead>
                        <span>{named.model}</span>
                        <span>{named.tier}</span>
                        <span className="text-right">{named.calls}</span>
                        <span className="text-right">{named.input}</span>
                        <span className="text-right">{named.cached}</span>
                        <span className="text-right">{named.output}</span>
                        <span className="text-right">{named.reasoning}</span>
                        <span className="text-right">{named.spent}</span>
                    </TableHead>
                    {rows.map((row) => (
                        <TableRow key={`${row.provider}/${row.model}/${row.tier}`}>
                            <TableCell mono={true} title={`${row.provider}/${row.model}`}>
                                {row.model}
                            </TableCell>
                            <TableCell muted={true}>
                                <span className={row.tier === "flex" ? "text-accent" : undefined}>
                                    {tierLabel(row.tier)}
                                </span>
                            </TableCell>
                            <TableCell mono={true} align="right">
                                {count(row.calls)}
                            </TableCell>
                            <TableCell mono={true} align="right">
                                {tokens(row.input)}
                            </TableCell>
                            <TableCell mono={true} align="right" muted={true}>
                                {tokens(row.cachedInput)}
                            </TableCell>
                            <TableCell mono={true} align="right">
                                {tokens(row.output)}
                            </TableCell>
                            <TableCell mono={true} align="right" muted={true}>
                                {tokens(row.reasoning)}
                            </TableCell>
                            <TableCell mono={true} align="right">
                                {usd(row.usd)}
                            </TableCell>
                        </TableRow>
                    ))}
                </DenseTable>
            </div>
            <p className="border-t border-hairline px-3 py-2 text-xs text-ink-dim">{named.note}</p>
        </Panel>
    );
}

export function SpendPanel(): ReactElement {
    const [days, setDays] = useState<SpendDays>(defaultSpendDays);
    const report = useSpendReport(days);
    const held = report.data;
    const busy = report.isPlaceholderData;
    const spentAnything = held !== undefined && held.totals.calls + held.totals.failed > 0;
    const failure = report.error === null ? null : react(report.error);
    const shownFailure =
        held === undefined && failure !== null && failure.kind !== "silent" && failure.kind !== "unlock"
            ? failure.message
            : null;

    return (
        <div className="flex flex-col gap-4">
            <Panel>
                <PanelHeader title={said.title}>
                    <Segmented
                        size="sm"
                        label={said.range}
                        value={String(days)}
                        options={rangeOptions}
                        onValueChange={(picked) => {
                            setDays(daysOf(picked));
                        }}
                    />
                </PanelHeader>
                {shownFailure !== null ? (
                    <div className="p-3">
                        <Banner
                            tone="danger"
                            title={shownFailure}
                            actions={
                                <Button size="sm" variant="secondary" onClick={() => void report.refetch()}>
                                    {copy.app.retry}
                                </Button>
                            }
                        />
                    </div>
                ) : held === undefined ? (
                    <div className="p-3">
                        <SkeletonRows rows={3} height={16} label={said.loading} />
                    </div>
                ) : spentAnything ? (
                    <Totals totals={held.totals} days={held.days} busy={busy} />
                ) : (
                    <div aria-busy={busy} className={cx("p-3 transition-opacity", busy && "opacity-60")}>
                        <EmptyState icon={MonitoringIcon} title={said.nothing(held.days)} body={said.nothingHint} />
                    </div>
                )}
            </Panel>
            {held !== undefined && spentAnything ? (
                <div aria-busy={busy} className={cx("flex flex-col gap-4 transition-opacity", busy && "opacity-60")}>
                    <PurposeTable rows={purposeRows(held.slices)} />
                    <ModelTable rows={modelRows(held.slices)} />
                </div>
            ) : null}
        </div>
    );
}
