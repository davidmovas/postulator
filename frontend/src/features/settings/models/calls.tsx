import type { ReactElement } from "react";
import { useMemo } from "react";

import { copy } from "../../../copy/index.js";
import { flatten } from "../../../data/call.js";
import { react } from "../../../data/errors.js";
import { useModelCalls } from "../../../data/hooks/models.js";
import type { ModelCall } from "../../../data/types.js";
import { absoluteTime, relativeTime, tokens, usd } from "../../../domain/format.js";
import {
    Banner,
    Button,
    DenseTable,
    Panel,
    PanelHeader,
    SkeletonRows,
    StatusBadge,
    TableHead,
    useWidthOf,
} from "../../../ui/index.js";
import type { ColumnLayout } from "../../../ui/collapse.js";
import { shown } from "../../../ui/collapse.js";
import type { CallColumn } from "./columns.js";
import { callLayout } from "./columns.js";
import { callStep, failureText, purposeLabel, tierLabel } from "./spend-labels.js";

const said = copy.settings.models.spend.recent;

export const recentPageSize = 25;

interface CallRowProps {
    call: ModelCall;
    layout: ColumnLayout<CallColumn>;
}

function CallRow({ call, layout }: CallRowProps): ReactElement {
    const step = callStep(call.purpose, call.step);
    const when = shown(layout, "when");
    return (
        <div
            role="row"
            style={{ gridTemplateColumns: layout.template }}
            className="grid items-center gap-x-3 gap-y-1 border-b border-inset px-3 py-1.5 text-sm last:border-b-0"
        >
            {when ? (
                <div role="cell" title={absoluteTime(call.createdAt)} className="truncate text-xs text-ink-dim">
                    {relativeTime(call.createdAt)}
                </div>
            ) : null}
            <div role="cell" title={when ? undefined : absoluteTime(call.createdAt)} className="min-w-0 truncate">
                <span className="text-ink">{purposeLabel(call.purpose)}</span>
                {step === "" ? null : <span className="text-ink-dim">{` · ${step}`}</span>}
            </div>
            <div
                role="cell"
                title={`${call.provider}/${call.model}`}
                className="min-w-0 truncate font-mono text-xs text-ink-soft"
            >
                {call.model}
            </div>
            {shown(layout, "tier") ? (
                <div role="cell" className="truncate text-xs">
                    <span className={call.tier === "flex" ? "text-accent" : "text-ink-dim"}>{tierLabel(call.tier)}</span>
                </div>
            ) : null}
            {shown(layout, "tokens") ? (
                <div
                    role="cell"
                    title={said.inside(tokens(call.cachedInput), tokens(call.reasoning))}
                    className="truncate text-right font-mono text-xs text-ink-soft"
                >
                    {said.flow(tokens(call.input), tokens(call.output))}
                </div>
            ) : null}
            <div role="cell" className="truncate text-right font-mono text-xs text-ink">
                {usd(call.usd)}
            </div>
            {call.status === "error" ? (
                <div role="cell" className="col-span-full flex min-w-0 items-start gap-2 text-xs">
                    <StatusBadge tone="danger">{said.failed}</StatusBadge>
                    <span className="min-w-0 pt-0.5 text-danger">{failureText(call.errorCode)}</span>
                </div>
            ) : null}
        </div>
    );
}

export function RecentCalls(): ReactElement {
    const listed = useModelCalls({}, recentPageSize);
    const [attach, width] = useWidthOf();
    const layout = useMemo(() => callLayout(width), [width]);
    const calls = flatten(listed.data?.pages);
    const failure = listed.error === null ? null : react(listed.error);
    const shownFailure =
        listed.data === undefined && failure !== null && failure.kind !== "silent" && failure.kind !== "unlock"
            ? failure.message
            : null;

    return (
        <Panel>
            <PanelHeader title={said.title} />
            {shownFailure !== null ? (
                <div className="p-3">
                    <Banner
                        tone="danger"
                        title={shownFailure}
                        actions={
                            <Button size="sm" variant="secondary" onClick={() => void listed.refetch()}>
                                {copy.app.retry}
                            </Button>
                        }
                    />
                </div>
            ) : listed.data === undefined ? (
                <div className="p-3">
                    <SkeletonRows rows={5} label={said.loading} />
                </div>
            ) : calls.length === 0 ? (
                <p className="p-3 text-xs text-ink-dim">{said.empty}</p>
            ) : (
                <div ref={attach} data-recent-calls={layout.columns.join(" ")} className="min-w-0">
                    <DenseTable columns={layout.template} label={said.title}>
                        <TableHead>
                            {shown(layout, "when") ? <span>{said.when}</span> : null}
                            <span>{said.what}</span>
                            <span>{said.model}</span>
                            {shown(layout, "tier") ? <span>{said.tier}</span> : null}
                            {shown(layout, "tokens") ? <span className="text-right">{said.tokens}</span> : null}
                            <span className="text-right">{said.spent}</span>
                        </TableHead>
                        {calls.map((call) => (
                            <CallRow key={call.id} call={call} layout={layout} />
                        ))}
                    </DenseTable>
                </div>
            )}
            {listed.hasNextPage ? (
                <div className="flex justify-center border-t border-hairline p-2">
                    <Button
                        size="sm"
                        variant="secondary"
                        busy={listed.isFetchingNextPage}
                        onClick={() => void listed.fetchNextPage()}
                    >
                        {copy.app.loadMore}
                    </Button>
                </div>
            ) : null}
        </Panel>
    );
}
