import type { ReactElement } from "react";
import { Link } from "react-router";

import { failure } from "../../data/errors.js";
import { usePageReport } from "../../data/hooks/reports.js";
import { copy } from "../../copy/index.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import {
    ChevronRightIcon,
    Panel,
    PanelHeader,
    Skeleton,
    StatusBadge,
    TaskAltIcon,
} from "../../ui/index.js";
import { itemStatusTone } from "./labels.js";

function fieldOf(held: unknown, key: string): unknown {
    if (typeof held !== "object" || held === null) {
        return undefined;
    }
    return (held as Record<string, unknown>)[key];
}

function scoreOf(held: unknown): number | null {
    const value = fieldOf(held, "score");
    return typeof value === "number" && Number.isFinite(value) ? value : null;
}

function textOf(held: unknown, key: string): string | null {
    const value = fieldOf(held, key);
    return typeof value === "string" && value !== "" ? value : null;
}

export interface PageReportPanelProps {
    pageId: string;
    siteId: string;
}

export function PageReportPanel({ pageId, siteId }: PageReportPanelProps): ReactElement {
    const report = usePageReport(pageId);

    if (report.isPending) {
        return (
            <Panel>
                <PanelHeader title={copy.pages.detail.report} />
                <div className="p-3">
                    <Skeleton height={16} width="70%" />
                </div>
            </Panel>
        );
    }

    if (report.isError) {
        const reported = failure(report.error);
        return (
            <Panel>
                <PanelHeader title={copy.pages.detail.report} />
                <div className="flex flex-col gap-1 p-3">
                    <p className="text-sm text-ink">
                        {reported.code === "NOT_FOUND" ? copy.pages.detail.noReport : reported.message}
                    </p>
                    {reported.code === "NOT_FOUND" ? (
                        <p className="text-xs text-ink-dim">{copy.pages.detail.noReportBody}</p>
                    ) : null}
                </div>
            </Panel>
        );
    }

    const held = report.data;
    if (held === undefined) {
        return (
            <Panel>
                <PanelHeader title={copy.pages.detail.report} />
                <p className="p-3 text-xs text-ink-dim">{copy.pages.detail.noReport}</p>
            </Panel>
        );
    }

    const judge = scoreOf(held.judge);
    const validation = scoreOf(held.validation);
    const publishStatus = textOf(held.publish, "status");
    const publishUrl = textOf(held.publish, "url");

    return (
        <Panel>
            <PanelHeader title={copy.pages.detail.report}>
                <StatusBadge tone={itemStatusTone(held.status)}>{held.status}</StatusBadge>
            </PanelHeader>
            <div className="flex flex-col gap-2 p-3">
                <Link
                    to={`/s/${siteId}/runs/${held.runId}/items/${held.itemId}`}
                    className="flex items-center gap-2 rounded-md border border-hairline bg-inset px-2.5 py-2 hover:bg-raised"
                >
                    <TaskAltIcon size={16} className="shrink-0" />
                    <div className="flex min-w-0 flex-1 flex-col">
                        <span className="truncate text-xs font-semibold">{copy.pages.detail.openRun}</span>
                        <span
                            className="truncate font-mono text-2xs text-ink-faint"
                            title={absoluteTime(held.finishedAt)}
                        >
                            {relativeTime(held.finishedAt)}
                        </span>
                    </div>
                    <ChevronRightIcon size={16} className="shrink-0 text-ink-dim" />
                </Link>
                <div className="flex flex-wrap gap-2">
                    {validation === null ? null : (
                        <StatusBadge tone="info" dot={false}>
                            {copy.pages.detail.validation(validation)}
                        </StatusBadge>
                    )}
                    {judge === null ? null : (
                        <StatusBadge tone="accent" dot={false}>
                            {copy.pages.detail.judge(judge)}
                        </StatusBadge>
                    )}
                    {publishStatus === null ? null : (
                        <StatusBadge tone="ok" dot={false}>
                            {copy.pages.detail.publishedAs(publishStatus)}
                        </StatusBadge>
                    )}
                </div>
                {publishUrl === null ? null : (
                    <div className="flex min-w-0 flex-col gap-0.5">
                        <span className="text-2xs tracking-label text-ink-faint uppercase">
                            {copy.pages.detail.liveUrl}
                        </span>
                        <span className="truncate font-mono text-xs text-ink-soft select-all">{publishUrl}</span>
                    </div>
                )}
            </div>
        </Panel>
    );
}
