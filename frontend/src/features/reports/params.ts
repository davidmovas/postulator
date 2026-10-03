import { choiceParam, queryCodec, textParam } from "../../data/params.js";

const reportTabs = ["site", "runs", "pages"] as const;

export type ReportTab = (typeof reportTabs)[number];

export interface ReportsQuery {
    tab: ReportTab;
    runId: string;
    pageId: string;
    prefix: string;
}

const codec = queryCodec<ReportsQuery>({
    tab: choiceParam("tab", reportTabs, "site"),
    runId: textParam("run"),
    prefix: textParam("prefix"),
    pageId: textParam("page"),
});

export const readQuery = codec.read;

export function writeQuery(query: ReportsQuery): URLSearchParams {
    return codec.write({
        tab: query.tab,
        runId: query.tab === "runs" ? query.runId : "",
        prefix: query.tab === "pages" ? query.prefix : "",
        pageId: query.tab === "pages" ? query.pageId : "",
    });
}

export function pathFilter(prefix: string): string {
    const trimmed = prefix.trim();
    if (trimmed === "") {
        return "";
    }
    return trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
}
