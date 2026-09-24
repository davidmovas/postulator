import type { ReactElement } from "react";
import { useMemo } from "react";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { react } from "../../data/errors.js";
import { useRegenerate, useRunItems } from "../../data/hooks/runs.js";
import type { RunEventsState } from "../../data/runs/log.js";
import type { Run } from "../../data/types.js";
import { Banner, Button, CloudSyncIcon, RefreshIcon } from "../../ui/index.js";
import type { RunView } from "./authority.js";
import { pauseReasonText, pauseReasonTone } from "./labels.js";
import { statusFailed } from "./statuses.js";

const failedPageSize = 200;

export interface RunNoticesProps {
    run: Run;
    view: RunView;
    events: RunEventsState;
    gap: boolean;
    failedItems: number;
    heldItems: number;
}

export function RunNotices({ run, view, events, gap, failedItems, heldItems }: RunNoticesProps): ReactElement | null {
    const logFailed = events.phase === "error" && events.error !== null;
    const failed = view.status === statusFailed && run.error !== "";
    const uncapped = !view.capped && !view.terminal;
    const shown = gap || logFailed || view.paused || failed || uncapped || failedItems > 0 || heldItems > 0;

    if (!shown) {
        return null;
    }

    return (
        <div className="flex shrink-0 flex-col gap-2 border-b border-hairline px-3 py-2">
            {gap ? (
                <Banner
                    tone="info"
                    icon={CloudSyncIcon}
                    title={copy.runs.reconnecting}
                    body={copy.runs.reconnectingBody}
                />
            ) : null}
            {logFailed ? <Banner tone="warn" title={copy.runs.logFailed} body={events.error?.message} /> : null}
            {view.budgetPaused ? (
                <Banner tone="danger" title={copy.runs.budgetPausedTitle} body={copy.runs.budgetPausedBody} />
            ) : view.paused ? (
                <Banner tone={pauseReasonTone(run.pauseReason)} title={pauseReasonText(run.pauseReason)} />
            ) : null}
            {failed ? <Banner tone="danger" title={copy.runs.detail.failure} body={run.error} /> : null}
            {heldItems > 0 ? (
                <Banner
                    tone="warn"
                    title={copy.runs.decisionTitle(heldItems)}
                    body={copy.runs.decisionBody}
                />
            ) : null}
            {failedItems > 0 ? <FailedItems runId={run.id} count={failedItems} /> : null}
            {uncapped ? <Banner tone="warn" title={copy.runs.detail.noCap} body={copy.runs.detail.noCapBody} /> : null}
        </div>
    );
}

function FailedItems({ runId, count }: { runId: string; count: number }): ReactElement {
    const listed = useRunItems(runId, statusFailed, failedPageSize);
    const regenerate = useRegenerate();
    const itemIds = useMemo(() => flatten(listed.data?.pages).map((item) => item.id), [listed.data]);
    const refused = regenerate.error === null ? null : react(regenerate.error);
    const refusal = refused === null || refused.kind === "silent" || refused.kind === "unlock" ? null : refused.message;

    return (
        <Banner
            tone="danger"
            title={copy.runs.failedTitle(count)}
            body={refusal ?? copy.runs.failedBody}
            actions={
                <Button
                    size="sm"
                    variant="primary"
                    data-run-regenerate-failed={true}
                    icon={RefreshIcon}
                    disabled={itemIds.length === 0}
                    busy={regenerate.isPending || listed.isPending}
                    onClick={() => {
                        regenerate.mutate({ runId, itemIds });
                    }}
                >
                    {copy.runs.regenerateFailed(itemIds.length === 0 ? count : itemIds.length)}
                </Button>
            }
        />
    );
}
