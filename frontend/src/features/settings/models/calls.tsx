import type { ReactElement } from "react";

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
} from "../../../ui/index.js";
import { callStep, failureText, purposeLabel, tierLabel } from "./spend-labels.js";

const said = copy.settings.models.spend.recent;

export const recentPageSize = 25;

const callColumns = "6.5rem minmax(9rem,1.4fr) minmax(8rem,1fr) 5rem 9rem 4.5rem";

function CallRow({ call }: { call: ModelCall }): ReactElement {
    const step = callStep(call.purpose, call.step);
    return (
        <div
            role="row"
            style={{ gridTemplateColumns: callColumns }}
            className="grid items-center gap-x-3 gap-y-1 border-b border-inset px-3 py-1.5 text-sm last:border-b-0"
        >
            <div role="cell" title={absoluteTime(call.createdAt)} className="truncate text-xs text-ink-dim">
                {relativeTime(call.createdAt)}
            </div>
            <div role="cell" className="min-w-0 truncate">
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
            <div role="cell" className="truncate text-xs">
                <span className={call.tier === "flex" ? "text-accent" : "text-ink-dim"}>{tierLabel(call.tier)}</span>
            </div>
            <div
                role="cell"
                title={said.inside(tokens(call.cachedInput), tokens(call.reasoning))}
                className="truncate text-right font-mono text-xs text-ink-soft"
            >
                {said.flow(tokens(call.input), tokens(call.output))}
            </div>
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
                <div className="overflow-x-auto">
                    <DenseTable columns={callColumns} label={said.title} className="min-w-[46rem]">
                        <TableHead>
                            <span>{said.when}</span>
                            <span>{said.what}</span>
                            <span>{said.model}</span>
                            <span>{said.tier}</span>
                            <span className="text-right">{said.tokens}</span>
                            <span className="text-right">{said.spent}</span>
                        </TableHead>
                        {calls.map((call) => (
                            <CallRow key={call.id} call={call} />
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
