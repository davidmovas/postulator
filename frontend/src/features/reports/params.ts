export const reportTabs = ["site", "runs", "pages"] as const;

export type ReportTab = (typeof reportTabs)[number];

export interface ReportsQuery {
    tab: ReportTab;
    runId: string;
    pageId: string;
    prefix: string;
}

export const defaultQuery: ReportsQuery = { tab: "site", runId: "", pageId: "", prefix: "" };

function isTab(value: string): value is ReportTab {
    return (reportTabs as readonly string[]).includes(value);
}

export function readQuery(params: URLSearchParams): ReportsQuery {
    const tab = params.get("tab") ?? "";
    return {
        tab: isTab(tab) ? tab : "site",
        runId: params.get("run") ?? "",
        pageId: params.get("page") ?? "",
        prefix: params.get("prefix") ?? "",
    };
}

export function writeQuery(query: ReportsQuery): URLSearchParams {
    const params = new URLSearchParams();
    if (query.tab !== defaultQuery.tab) {
        params.set("tab", query.tab);
    }
    if (query.tab === "runs" && query.runId !== "") {
        params.set("run", query.runId);
    }
    if (query.tab === "pages") {
        if (query.prefix !== "") {
            params.set("prefix", query.prefix);
        }
        if (query.pageId !== "") {
            params.set("page", query.pageId);
        }
    }
    return params;
}

export function pathFilter(prefix: string): string {
    const trimmed = prefix.trim();
    if (trimmed === "") {
        return "";
    }
    return trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
}
