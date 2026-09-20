import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { RunEventsState } from "../../data/runs/log.js";
import type { Run } from "../../data/types.js";
import { Banner, CloudSyncIcon } from "../../ui/index.js";
import type { RunView } from "./authority.js";
import { pauseReasonText, pauseReasonTone } from "./labels.js";
import { statusFailed } from "./statuses.js";

export interface RunNoticesProps {
    run: Run;
    view: RunView;
    events: RunEventsState;
    gap: boolean;
}

export function RunNotices({ run, view, events, gap }: RunNoticesProps): ReactElement | null {
    const logFailed = events.phase === "error" && events.error !== null;
    const failed = view.status === statusFailed && run.error !== "";
    const uncapped = !view.capped && !view.terminal;
    const shown = gap || logFailed || view.paused || failed || uncapped;

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
            {uncapped ? <Banner tone="warn" title={copy.runs.detail.noCap} body={copy.runs.detail.noCapBody} /> : null}
        </div>
    );
}
