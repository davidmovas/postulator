import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../copy/index.js";
import { useSyncSite } from "../../data/hooks/sync.js";
import type { Site } from "../../data/types.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import { Button, Checkbox, SectionLabel, SyncIcon } from "../../ui/index.js";

const said = copy.sites.sync;

export interface SiteSyncPanelProps {
    site: Site;
}

export function SiteSyncPanel({ site }: SiteSyncPanelProps): ReactElement {
    const sync = useSyncSite();
    const [full, setFull] = useState(false);
    const [renderRescan, setRenderRescan] = useState(false);
    const observation = site.settings.observation;

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
            <span title={said.fullTooltip} className="w-fit">
                <Checkbox
                    label={said.full}
                    checked={full}
                    onChange={(event) => {
                        setFull(event.currentTarget.checked);
                    }}
                />
            </span>
            <span title={said.rescanTooltip} className="w-fit">
                <Checkbox
                    label={said.rescan}
                    checked={renderRescan}
                    onChange={(event) => {
                        setRenderRescan(event.currentTarget.checked);
                    }}
                />
            </span>
            <Button
                variant="secondary"
                icon={SyncIcon}
                busy={sync.isPending}
                onClick={() => {
                    sync.mutate(
                        { siteId: site.id, full, renderRescan },
                        {
                            onSuccess: () => {
                                setFull(false);
                                setRenderRescan(false);
                            },
                        },
                    );
                }}
            >
                {said.start}
            </Button>
        </section>
    );
}
