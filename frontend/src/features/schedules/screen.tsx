import type { ReactElement } from "react";
import { useMemo } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { react } from "../../data/errors.js";
import { useGraph } from "../../data/hooks/graph.js";
import { useRuns } from "../../data/hooks/runs.js";
import { useDisableSchedule, useEnableSchedule, useSchedules } from "../../data/hooks/schedules.js";
import { useTemplates } from "../../data/hooks/templates.js";
import type { Run, Schedule } from "../../data/types.js";
import {
    AddIcon,
    Banner,
    Button,
    CountBadge,
    EmptyState,
    ScheduleIcon,
    Screen,
    Segmented,
    SkeletonRows,
    Toolbar,
} from "../../ui/index.js";
import { SchedulePanel } from "./panel.js";
import type { EnabledFilter, SchedulesQuery } from "./params.js";
import { enabledOf, readQuery, wantsNew, writeQuery } from "./params.js";
import { SchedulesTable } from "./table.js";

const runsPage = 100;

export function SchedulesScreen(): ReactElement {
    const params = useParams();
    const siteId = params.siteId ?? "";
    const navigate = useNavigate();
    const [searchParams, setSearchParams] = useSearchParams();
    const query = useMemo(() => readQuery(searchParams), [searchParams]);
    const creating = wantsNew(searchParams);

    const listed = useSchedules({ siteId, enabled: enabledOf(query.show) });
    const graph = useGraph(siteId === "" ? null : siteId);
    const templates = useTemplates({ siteId });
    const runs = useRuns({ siteId }, null, runsPage);
    const enable = useEnableSchedule();
    const disable = useDisableSchedule();

    const rows = useMemo(() => flatten(listed.data?.pages), [listed.data]);
    const templateRows = useMemo(() => flatten(templates.data?.pages), [templates.data]);
    const entities = useMemo(() => graph.data?.entities ?? [], [graph.data]);
    const entityNames = useMemo(() => {
        const held = new Map<string, string>();
        for (const entity of entities) {
            held.set(entity.id, entity.name);
        }
        return held;
    }, [entities]);
    const lastRuns = useMemo(() => {
        const held = new Map<string, Run>();
        for (const run of flatten(runs.data?.pages)) {
            held.set(run.id, run);
        }
        return held;
    }, [runs.data]);

    const selected = rows.find((row) => row.id === query.id) ?? null;
    const failure = listed.error === null ? null : react(listed.error);

    const change = (next: SchedulesQuery, createNew = false): void => {
        const written = writeQuery(next);
        if (createNew) {
            written.set("action", "new");
        }
        setSearchParams(written, { replace: true });
    };

    const openPanel = (schedule: Schedule): void => {
        change({ ...query, id: schedule.id });
    };

    const toggle = (schedule: Schedule): void => {
        if (schedule.enabled) {
            disable.mutate({ id: schedule.id });
        } else {
            enable.mutate({ id: schedule.id });
        }
    };

    const panel =
        creating || selected !== null ? (
            <SchedulePanel
                key={creating ? "new" : (selected?.id ?? "")}
                siteId={siteId}
                schedule={creating ? null : selected}
                templates={templateRows}
                entities={entities}
                onClose={() => {
                    change({ ...query, id: "" });
                }}
                onCreated={openPanel}
                onStarted={(runId) => {
                    void navigate(`/s/${siteId}/runs/${runId}`);
                }}
                onDeleted={() => {
                    change({ ...query, id: "" });
                }}
            />
        ) : undefined;

    return (
        <Screen
            title={copy.schedules.title}
            badge={listed.isPending ? undefined : <CountBadge tone="muted" count={rows.length} />}
            actions={
                <Button
                    variant="primary"
                    icon={AddIcon}
                    onClick={() => {
                        change({ ...query, id: "" }, true);
                    }}
                >
                    {copy.schedules.create}
                </Button>
            }
            toolbar={
                <Toolbar label={copy.schedules.filter.label}>
                    <Segmented<EnabledFilter>
                        label={copy.schedules.filter.label}
                        value={query.show}
                        options={[
                            { value: "all", label: copy.schedules.filter.all },
                            { value: "on", label: copy.schedules.filter.on },
                            { value: "off", label: copy.schedules.filter.off },
                        ]}
                        onValueChange={(show) => {
                            change({ ...query, show });
                        }}
                    />
                </Toolbar>
            }
            variant="split"
            right={panel}
        >
            {listed.isPending ? (
                <div className="p-4">
                    <SkeletonRows rows={8} label={copy.schedules.loading} />
                </div>
            ) : failure !== null && failure.kind !== "silent" && failure.kind !== "unlock" ? (
                <div className="p-4">
                    <Banner
                        tone="danger"
                        title={failure.message}
                        actions={
                            <Button size="sm" variant="secondary" onClick={() => void listed.refetch()}>
                                {copy.app.retry}
                            </Button>
                        }
                    />
                </div>
            ) : rows.length === 0 ? (
                <div className="p-4">
                    <EmptyState
                        icon={ScheduleIcon}
                        title={copy.empty.schedules}
                        actions={
                            <Button
                                variant="primary"
                                icon={AddIcon}
                                onClick={() => {
                                    change({ ...query, id: "" }, true);
                                }}
                            >
                                {copy.schedules.create}
                            </Button>
                        }
                    />
                </div>
            ) : (
                <div className="min-h-0 flex-1 overflow-auto">
                    <SchedulesTable
                        siteId={siteId}
                        rows={rows}
                        entityNames={entityNames}
                        lastRuns={lastRuns}
                        selectedId={creating ? "" : query.id}
                        onSelect={(id) => {
                            change({ ...query, id });
                        }}
                        onToggle={toggle}
                    />
                </div>
            )}
        </Screen>
    );
}
