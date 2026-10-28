import type { ReactElement } from "react";
import { useState } from "react";
import { useNavigate } from "react-router";

import { copy } from "../../copy/index.js";
import { useActiveSync, useLatestSync } from "../../data/hooks/sync.js";
import type { Site } from "../../data/types.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import { Checkbox, SectionLabel } from "../../ui/index.js";
import { SyncButton, SyncLive, SyncStopped, syncRunPath } from "../sync/index.js";

const said = copy.sites.sync;

const failedStatus = "failed";

export interface SiteSyncPanelProps {
    site: Site;
}

export function SiteSyncPanel({ site }: SiteSyncPanelProps): ReactElement {
    const navigate = useNavigate();
    const active = useActiveSync(site.id);
    const latest = useLatestSync(site.id);
    const [full, setFull] = useState(false);
    const [renderRescan, setRenderRescan] = useState(false);
    const observation = site.settings.observation;
    const stopped = active === null && latest.progress?.status === failedStatus ? latest.progress : null;

    return (
        <section aria-label={said.title} className="flex flex-col gap-2">
            <SectionLabel>{said.title}</SectionLabel>
            <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
                <dt className="text-ink-faint">{said.lastWalk}</dt>
                <dd className="text-ink-soft" title={absoluteTime(observation.lastFinishedAt)}>
                    {relativeTime(observation.lastFinishedAt)}
                </dd>
                <dt className="text-ink-faint">{said.lastFullWalk}</dt>
                <dd className="text-ink-soft" title={absoluteTime(observation.lastFullWalkAt)}>
                    {relativeTime(observation.lastFullWalkAt)}
                </dd>
            </dl>
            {active === null ? null : <SyncLive progress={active} />}
            {stopped === null ? null : (
                <SyncStopped
                    runId={stopped.runId}
                    stage={stopped.stage}
                    error={stopped.error}
                    compact={true}
                    onOpen={(runId) => {
                        void navigate(syncRunPath(site.id, runId));
                    }}
                />
            )}
            <span title={said.fullTooltip} className="w-fit">
                <Checkbox
                    label={said.full}
                    checked={full}
                    disabled={active !== null}
                    onChange={(event) => {
                        setFull(event.currentTarget.checked);
                    }}
                />
            </span>
            <span title={said.rescanTooltip} className="w-fit">
                <Checkbox
                    label={said.rescan}
                    checked={renderRescan}
                    disabled={active !== null}
                    onChange={(event) => {
                        setRenderRescan(event.currentTarget.checked);
                    }}
                />
            </span>
            <span className="flex min-w-0">
                <SyncButton
                    siteId={site.id}
                    label={said.start}
                    options={{ full, renderRescan }}
                    onStarted={() => {
                        setFull(false);
                        setRenderRescan(false);
                    }}
                />
            </span>
        </section>
    );
}
