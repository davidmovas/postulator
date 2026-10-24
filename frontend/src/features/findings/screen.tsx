import type { ReactElement } from "react";
import { useMemo, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { useFindingCounts, useFindings, useResolveFinding } from "../../data/hooks/findings.js";
import type { Finding, FindingChoice, SyncSiteRequest } from "../../data/types.js";
import { CountBadge, Screen } from "../../ui/index.js";
import { cardsSearch } from "../graph/model/params.js";
import { StartRunDrawer } from "../runs/start.js";
import type { StartSeed } from "../runs/start-seed.js";
import { seedOf } from "../runs/start-seed.js";
import { SyncDialog } from "../sites/sync-dialog.js";
import { BulkDialog } from "./bulk-dialog.js";
import { FindingList } from "./list.js";
import { bulkOptions } from "./model/bulk.js";
import { countsOf, defaultQuery, filterOf, narrowed, readQuery, writeQuery } from "./model/params.js";
import type { FindingsQuery } from "./model/params.js";
import { FindingsRail } from "./rail.js";
import type { Resolving } from "./resolve-dialog.js";
import { ResolveDialog } from "./resolve-dialog.js";
import { onTheGraph } from "./row.js";
import { FindingsToolbar } from "./toolbar.js";

const pageSize = 100;
const runKind = "run";

export function FindingsScreen(): ReactElement {
    const params = useParams();
    const navigate = useNavigate();
    const [searchParams, setSearchParams] = useSearchParams();
    const siteId = params.siteId ?? "";
    const query = useMemo(() => readQuery(searchParams), [searchParams]);
    const filter = useMemo(() => filterOf(siteId, query), [siteId, query]);
    const listed = useFindings(filter, pageSize);
    const counts = useFindingCounts(countsOf(siteId, query));
    const findings = useMemo(() => flatten(listed.data?.pages), [listed.data]);
    const prepare = useResolveFinding();
    const [resolving, setResolving] = useState<Resolving | null>(null);
    const [bulk, setBulk] = useState<string | null>(null);
    const [seed, setSeed] = useState<StartSeed | null>(null);
    const [sync, setSync] = useState<SyncSiteRequest | null>(null);

    const offered = useMemo(() => bulkOptions(query.code, query.status, findings), [query.code, query.status, findings]);
    const codeCount = counts.data?.codes?.find((held) => held.code === query.code)?.count ?? 0;

    const change = (next: FindingsQuery): void => {
        setSearchParams(writeQuery(next), { replace: true });
    };

    const reread = (): void => {
        void listed.refetch();
        void counts.refetch();
    };

    const toRun = (runId: string): void => {
        setSeed(null);
        setSync(null);
        void navigate(`/s/${siteId}/runs/${runId}`);
    };

    const choose = (finding: Finding, option: FindingChoice): void => {
        if (onTheGraph(finding, option.name)) {
            void navigate(`/s/${siteId}/graph${cardsSearch(finding.entityId ?? "")}`);
            return;
        }
        if (option.kind !== runKind) {
            setResolving({ finding, option: option.name });
            return;
        }
        prepare.mutate(
            { siteId, findingId: finding.id, option: option.name },
            {
                onSuccess: (answered) => {
                    if (answered.start !== undefined && answered.start !== null) {
                        setSeed(seedOf(answered.start));
                    } else if (answered.sync !== undefined && answered.sync !== null) {
                        setSync(answered.sync);
                    }
                },
            },
        );
    };

    return (
        <Screen
            title={copy.findings.title}
            badge={counts.data === undefined ? undefined : <CountBadge tone="muted" count={counts.data.total} />}
            variant="split"
            toolbar={
                <FindingsToolbar
                    siteId={siteId}
                    query={query}
                    bulk={{ options: codeCount === 0 ? [] : offered, count: codeCount, onBulk: setBulk }}
                    onChange={change}
                />
            }
            left={
                <FindingsRail
                    counts={counts.data}
                    code={query.code}
                    onCode={(code) => {
                        change({ ...query, code });
                    }}
                />
            }
        >
            <FindingList
                findings={findings}
                pending={listed.isPending}
                error={listed.error}
                open={query.status === "open"}
                narrowed={narrowed(query)}
                hasMore={listed.hasNextPage}
                onMore={() => {
                    void listed.fetchNextPage();
                }}
                onRetry={reread}
                onReset={() => {
                    change({ ...defaultQuery, status: query.status });
                }}
                onOpenPage={(pageId) => {
                    void navigate(`/s/${siteId}/pages/${pageId}`);
                }}
                onChoose={choose}
            />
            <ResolveDialog
                siteId={siteId}
                resolving={resolving}
                onClose={() => {
                    setResolving(null);
                }}
                onGone={reread}
            />
            <BulkDialog
                siteId={siteId}
                code={query.code}
                filter={filter}
                option={bulk}
                onClose={() => {
                    setBulk(null);
                }}
                onGone={reread}
            />
            <StartRunDrawer
                open={seed !== null}
                onOpenChange={(next) => {
                    if (!next) {
                        setSeed(null);
                    }
                }}
                siteId={siteId}
                seed={seed}
                onStarted={toRun}
            />
            <SyncDialog
                request={sync}
                onOpenChange={(next) => {
                    if (!next) {
                        setSync(null);
                    }
                }}
                onStarted={toRun}
            />
        </Screen>
    );
}
