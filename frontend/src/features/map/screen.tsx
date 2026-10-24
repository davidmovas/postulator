import type { ReactElement } from "react";
import { useMemo } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";

import { copy } from "../../copy/index.js";
import { useSyncSite } from "../../data/hooks/sync.js";
import { AccountTreeIcon, Button, EmptyState, Screen, UploadFileIcon } from "../../ui/index.js";
import type { PageTab } from "../pages/index.js";
import { PageDrawer, readTab, useEntityIndex, withTab } from "../pages/index.js";
import { SubtreeCrumbs } from "./crumbs.js";
import { FlatList } from "./flat.js";
import { defaultQuery, filterOf, flat, readQuery, searchOf, writeQuery } from "./model/params.js";
import type { MapQuery } from "./model/params.js";
import { MapToolbar } from "./toolbar.js";
import { MapTree } from "./tree.js";

const said = copy.map;

export function SiteMapScreen(): ReactElement {
    const params = useParams();
    const navigate = useNavigate();
    const [searchParams, setSearchParams] = useSearchParams();
    const siteId = params.siteId ?? "";
    const pageId = params.pageId ?? null;
    const query = useMemo(() => readQuery(searchParams), [searchParams]);
    const filter = useMemo(() => filterOf(siteId, query), [siteId, query]);
    const search = searchOf(query);
    const index = useEntityIndex(siteId);
    const sync = useSyncSite();

    const change = (next: MapQuery): void => {
        setSearchParams(writeQuery(next), { replace: true });
    };

    const open = (id: string): void => {
        void navigate(`/s/${siteId}/map/${id}${search}`);
    };

    const under = (id: string): void => {
        change({ ...defaultQuery, under: id });
    };

    return (
        <Screen
            title={said.title}
            variant="full"
            toolbar={<MapToolbar siteId={siteId} query={query} onChange={change} />}
        >
            {query.under === "" ? null : <SubtreeCrumbs siteId={siteId} under={query.under} onUnder={under} />}
            {flat(query) ? (
                <FlatList
                    siteId={siteId}
                    filter={filter}
                    onOpen={open}
                    onUnder={under}
                    onReset={() => {
                        change({ ...defaultQuery, under: query.under });
                    }}
                />
            ) : (
                <MapTree
                    key={siteId}
                    siteId={siteId}
                    root={query.under}
                    onOpen={open}
                    onUnder={under}
                    empty={
                        <EmptyState
                            icon={AccountTreeIcon}
                            title={said.empty.title}
                            body={said.empty.body}
                            className="w-96"
                            actions={
                                <>
                                    <Button
                                        variant="primary"
                                        busy={sync.isPending}
                                        onClick={() => {
                                            sync.mutate({ siteId });
                                        }}
                                    >
                                        {sync.isPending ? said.empty.syncing : said.empty.sync}
                                    </Button>
                                    <Button
                                        icon={UploadFileIcon}
                                        onClick={() => {
                                            void navigate(`/s/${siteId}/import`);
                                        }}
                                    >
                                        {said.empty.import}
                                    </Button>
                                </>
                            }
                        />
                    }
                />
            )}
            {pageId === null ? null : (
                <PageDrawer
                    key={pageId}
                    pageId={pageId}
                    siteId={siteId}
                    index={index}
                    search=""
                    tab={readTab(searchParams)}
                    onTabChange={(tab: PageTab) => {
                        setSearchParams(withTab(searchParams, tab), { replace: true });
                    }}
                    onClose={() => {
                        void navigate(`/s/${siteId}/map${search}`);
                    }}
                />
            )}
        </Screen>
    );
}
