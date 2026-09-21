import type { ReactElement } from "react";
import { useMemo } from "react";
import { Link } from "react-router";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { failure } from "../../data/errors.js";
import { usePages } from "../../data/hooks/pages.js";
import { usePageReport } from "../../data/hooks/reports.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import {
    Banner,
    DescriptionIcon,
    EmptyState,
    Field,
    Input,
    SearchIcon,
    SkeletonRows,
    StatusBadge,
    TaskAltIcon,
} from "../../ui/index.js";
import { statusTone } from "../pages/labels.js";
import { statusLabel } from "../runs/labels.js";
import { PageReportCards } from "./page-cards.js";
import { pathFilter } from "./params.js";

const matchLimit = 12;

export interface PagesTabProps {
    siteId: string;
    prefix: string;
    pageId: string;
    onPrefix: (prefix: string) => void;
    onSelect: (pageId: string) => void;
}

export function PagesTab({ siteId, prefix, pageId, onPrefix, onSelect }: PagesTabProps): ReactElement {
    const filter = pathFilter(prefix);
    const listed = usePages(
        filter === "" ? { siteId } : { siteId, pathPrefix: filter },
        { field: "path", desc: false },
        matchLimit,
    );
    const matches = useMemo(() => flatten(listed.data?.pages).slice(0, matchLimit), [listed.data]);
    const report = usePageReport(pageId === "" ? null : pageId);
    const selected = matches.find((page) => page.id === pageId);
    const reported = report.error === null ? null : failure(report.error);

    return (
        <div className="flex min-h-0 flex-1 gap-4 overflow-auto p-4">
            <div className="flex w-72 shrink-0 flex-col gap-2">
                <Field label={copy.reports.pages.search}>
                    {(control) => (
                        <Input
                            id={control.id}
                            mono={true}
                            value={prefix}
                            placeholder={copy.reports.pages.searchPlaceholder}
                            onChange={(event) => {
                                onPrefix(event.target.value);
                            }}
                        />
                    )}
                </Field>
                {listed.isPending ? (
                    <SkeletonRows rows={6} label={copy.reports.pages.search} />
                ) : matches.length === 0 ? (
                    <EmptyState icon={SearchIcon} title={copy.reports.pages.noMatch} />
                ) : (
                    <ul className="flex flex-col overflow-hidden rounded-lg border border-hairline">
                        {matches.map((page) => (
                            <li key={page.id} className="border-b border-hairline last:border-b-0">
                                <button
                                    type="button"
                                    data-page-id={page.id}
                                    title={page.path}
                                    onClick={() => {
                                        onSelect(page.id);
                                    }}
                                    className={
                                        page.id === pageId
                                            ? "flex h-7 w-full items-center gap-2 bg-raised px-3 text-left"
                                            : "flex h-7 w-full items-center gap-2 px-3 text-left hover:bg-raised"
                                    }
                                >
                                    <DescriptionIcon size={13} className="shrink-0 text-ink-faint" />
                                    <span className="min-w-0 flex-1 truncate font-mono text-xs text-ink">
                                        {page.path}
                                    </span>
                                    <StatusBadge tone={statusTone(page.status)} dot={false}>
                                        {page.status}
                                    </StatusBadge>
                                </button>
                            </li>
                        ))}
                    </ul>
                )}
            </div>
            <div className="flex min-w-0 flex-1 flex-col gap-3">
                {pageId === "" ? (
                    <EmptyState icon={SearchIcon} title={copy.reports.pages.searchHint} />
                ) : report.isPending ? (
                    <SkeletonRows rows={8} label={copy.reports.loading} />
                ) : reported !== null && reported.code === "NOT_FOUND" ? (
                    <EmptyState icon={TaskAltIcon} title={copy.reports.pages.noReport} />
                ) : reported !== null ? (
                    <Banner tone="danger" title={reported.message} />
                ) : report.data === undefined ? (
                    <SkeletonRows rows={8} label={copy.reports.loading} />
                ) : (
                    <>
                        <header className="flex flex-wrap items-center gap-3">
                            <h2 className="min-w-0 truncate font-mono text-sm text-ink" title={report.data.path}>
                                {report.data.path}
                            </h2>
                            <StatusBadge tone={statusTone(selected?.status ?? "")}>
                                {statusLabel(report.data.status)}
                            </StatusBadge>
                            <span
                                className="text-2xs text-ink-faint"
                                title={absoluteTime(report.data.finishedAt ?? null)}
                            >
                                {copy.reports.pages.finishedAt} {relativeTime(report.data.finishedAt ?? null)}
                            </span>
                            <Link
                                to={`/s/${siteId}/pages/${report.data.pageId}`}
                                className="text-2xs text-accent hover:underline"
                            >
                                {copy.reports.pages.openPage}
                            </Link>
                            <Link
                                to={`/s/${siteId}/runs/${report.data.runId}/items/${report.data.itemId}`}
                                className="text-2xs text-accent hover:underline"
                            >
                                {copy.reports.pages.openItem}
                            </Link>
                        </header>
                        <PageReportCards report={report.data} />
                    </>
                )}
            </div>
        </div>
    );
}
