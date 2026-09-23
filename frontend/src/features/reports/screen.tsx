import type { ReactElement } from "react";
import { useMemo } from "react";
import { useParams, useSearchParams } from "react-router";

import { copy } from "../../copy/index.js";
import { Screen, Tabs } from "../../ui/index.js";
import { PagesTab } from "./pages-tab.js";
import type { ReportTab, ReportsQuery } from "./params.js";
import { readQuery, writeQuery } from "./params.js";
import { RunReportPanel } from "./run-panel.js";
import { RunsTab } from "./runs-tab.js";
import { SiteTab } from "./site-tab.js";

export function ReportsScreen(): ReactElement {
    const params = useParams();
    const siteId = params.siteId ?? "";
    const [searchParams, setSearchParams] = useSearchParams();
    const query = useMemo(() => readQuery(searchParams), [searchParams]);

    const change = (next: ReportsQuery): void => {
        setSearchParams(writeQuery(next), { replace: true });
    };

    return (
        <Screen
            title={copy.reports.title}
            tabs={
                <Tabs<ReportTab>
                    label={copy.reports.tabs.label}
                    value={query.tab}
                    items={[
                        { key: "site", label: copy.reports.tabs.site },
                        { key: "runs", label: copy.reports.tabs.runs },
                        { key: "pages", label: copy.reports.tabs.pages },
                    ]}
                    onValueChange={(tab) => {
                        change({ ...query, tab });
                    }}
                />
            }
            variant="split"
            right={
                query.tab === "runs" ? <RunReportPanel siteId={siteId} runId={query.runId} /> : undefined
            }
        >
            {query.tab === "site" ? (
                <SiteTab siteId={siteId} />
            ) : query.tab === "runs" ? (
                <RunsTab
                    siteId={siteId}
                    selectedId={query.runId}
                    onSelect={(runId) => {
                        change({ ...query, runId });
                    }}
                />
            ) : (
                <PagesTab
                    siteId={siteId}
                    prefix={query.prefix}
                    pageId={query.pageId}
                    onPrefix={(prefix) => {
                        change({ ...query, prefix, pageId: "" });
                    }}
                    onSelect={(pageId) => {
                        change({ ...query, pageId });
                    }}
                />
            )}
        </Screen>
    );
}
