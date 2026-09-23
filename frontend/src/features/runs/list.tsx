import type { ReactElement } from "react";
import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { failure } from "../../data/errors.js";
import { useRuns } from "../../data/hooks/runs.js";
import { activeRunStatuses } from "../../generated/vocab.js";
import {
    Banner,
    Button,
    CountBadge,
    EmptyState,
    FilterAltIcon,
    HistoryIcon,
    PlayArrowIcon,
    Screen,
    SkeletonRows,
} from "../../ui/index.js";
import { useNow } from "./clock.js";
import { RunFilters } from "./filters.js";
import { defaultQuery, filterOf, narrowed, nextSort, readQuery, searchOf, wantsNew, writeQuery } from "./params.js";
import type { RunsQuery } from "./params.js";
import { RunTable } from "./run-table.js";
import { StartRunDrawer } from "./start.js";

const pageSize = 100;
const tickMs = 5000;

export function RunsScreen(): ReactElement {
    const params = useParams();
    const navigate = useNavigate();
    const [searchParams, setSearchParams] = useSearchParams();
    const siteId = params.siteId ?? "";
    const query = useMemo(() => readQuery(searchParams), [searchParams]);
    const search = searchOf(query);
    const filter = useMemo(() => filterOf(siteId, query), [siteId, query]);
    const listed = useRuns(filter, query.sort, pageSize);
    const rows = useMemo(() => flatten(listed.data?.pages), [listed.data]);
    const [starting, setStarting] = useState(false);
    const asked = wantsNew(searchParams);
    const active = rows.some((run) => (activeRunStatuses as readonly string[]).includes(run.status));
    const now = useNow(tickMs, active);

    useEffect(() => {
        if (asked) {
            setStarting(true);
            setSearchParams(writeQuery(readQuery(searchParams)), { replace: true });
        }
    }, [asked, searchParams, setSearchParams]);

    const change = (next: RunsQuery): void => {
        setSearchParams(writeQuery(next), { replace: true });
    };

    const start = (
        <Button
            data-run-start={true}
            variant="primary"
            icon={PlayArrowIcon}
            onClick={() => {
                setStarting(true);
            }}
        >
            {copy.runs.start.open}
        </Button>
    );

    return (
        <Screen
            title={copy.runs.title}
            badge={rows.length === 0 ? undefined : <CountBadge tone="muted" count={rows.length} />}
            variant="split"
            actions={start}
            left={<RunFilters query={query} onChange={change} />}
        >
            {listed.isPending ? (
                <div className="min-h-0 flex-1 overflow-auto p-3">
                    <SkeletonRows rows={10} label={copy.runs.loading} />
                </div>
            ) : listed.isError ? (
                <div className="min-h-0 flex-1 overflow-auto p-4">
                    <Banner
                        tone="danger"
                        title={failure(listed.error).message}
                        className="max-w-lg"
                        actions={
                            <Button
                                onClick={() => {
                                    void listed.refetch();
                                }}
                            >
                                {copy.app.retry}
                            </Button>
                        }
                    />
                </div>
            ) : rows.length === 0 ? (
                <div className="flex min-h-0 flex-1 items-start justify-center overflow-auto p-6">
                    {narrowed(query) ? (
                        <EmptyState
                            icon={FilterAltIcon}
                            title={copy.runs.noMatch}
                            actions={
                                <Button
                                    onClick={() => {
                                        change({ ...defaultQuery, sort: query.sort });
                                    }}
                                >
                                    {copy.runs.filters.reset}
                                </Button>
                            }
                        />
                    ) : (
                        <EmptyState
                            icon={HistoryIcon}
                            title={copy.runs.title}
                            body={copy.empty.runs}
                            className="w-96"
                            actions={start}
                        />
                    )}
                </div>
            ) : (
                <RunTable
                    rows={rows}
                    now={now}
                    sort={query.sort}
                    selectedId={null}
                    scrollKey={`${siteId}:runs${search}`}
                    onSortChange={(field) => {
                        change({ ...query, sort: nextSort(query.sort, field) });
                    }}
                    onOpen={(runId) => {
                        void navigate(`/s/${siteId}/runs/${runId}`);
                    }}
                    footer={
                        listed.hasNextPage ? (
                            <div className="flex justify-center border-t border-hairline p-2">
                                <Button
                                    size="sm"
                                    busy={listed.isFetchingNextPage}
                                    onClick={() => {
                                        void listed.fetchNextPage();
                                    }}
                                >
                                    {copy.app.loadMore}
                                </Button>
                            </div>
                        ) : null
                    }
                />
            )}
            <StartRunDrawer
                open={starting}
                onOpenChange={setStarting}
                siteId={siteId}
                onStarted={(runId) => {
                    setStarting(false);
                    void navigate(`/s/${siteId}/runs/${runId}`);
                }}
            />
        </Screen>
    );
}
