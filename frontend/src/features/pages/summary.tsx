import type { ReactElement, ReactNode } from "react";
import { Link } from "react-router";

import { copy } from "../../copy/index.js";
import { failure } from "../../data/errors.js";
import { useEntity } from "../../data/hooks/graph.js";
import { usePage } from "../../data/hooks/pages.js";
import { usePageReport } from "../../data/hooks/reports.js";
import { useSite } from "../../data/hooks/sites.js";
import { isBrowsable, openExternal } from "../../data/host.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import {
    Button,
    ChevronRightIcon,
    EditNoteIcon,
    IconButton,
    LinkIcon,
    LinkOffIcon,
    OpenInNewIcon,
    SectionLabel,
    SkeletonRows,
    StatusBadge,
    SyncProblemIcon,
    TaskAltIcon,
    VerifiedIcon,
} from "../../ui/index.js";
import { entityIcon, statusTone } from "./labels.js";

const shownLinks = 6;

export function liveUrl(baseUrl: string, path: string): string {
    if (baseUrl === "") {
        return "";
    }
    return `${baseUrl.replace(/\/+$/, "")}${path}`;
}

function Row({ label, children }: { label: string; children: ReactNode }): ReactElement {
    return (
        <div className="flex items-baseline justify-between gap-3">
            <dt className="shrink-0 text-2xs font-semibold tracking-label text-ink-faint uppercase">{label}</dt>
            <dd className="min-w-0 truncate text-sm text-ink">{children}</dd>
        </div>
    );
}

export interface PageSummaryProps {
    pageId: string;
    siteId: string;
    search: string;
    onOpen: (pageId: string) => void;
}

export function PageSummary({ pageId, siteId, search, onOpen }: PageSummaryProps): ReactElement {
    const detail = usePage(pageId);
    const site = useSite(siteId);
    const page = detail.data?.page;
    const entity = useEntity(page?.entityId ?? null);
    const report = usePageReport(pageId);

    if (detail.isPending) {
        return (
            <div className="p-3">
                <SkeletonRows rows={8} label={copy.app.loading} />
            </div>
        );
    }

    if (page === undefined) {
        return (
            <div className="p-3">
                <p className="text-sm text-ink">
                    {failure(detail.error).code === "NOT_FOUND"
                        ? copy.pages.detail.notFound
                        : failure(detail.error).message}
                </p>
            </div>
        );
    }

    const links = detail.data?.links ?? [];
    const generated = links.filter((held) => held.origin === "generated").length;
    const url = liveUrl(site.data?.site.baseUrl ?? "", page.path);
    const reachable = isBrowsable(url) && page.wpId !== null && page.status !== "archived";
    const Icon = entityIcon(entity.data?.entity.kind ?? "");
    const canonical = entity.data?.entity.canonicalPageId === page.id;

    return (
        <div className="flex h-full min-h-0 flex-col">
            <header className="flex h-8 shrink-0 items-center justify-between gap-2 border-b border-hairline px-3">
                <h2 className="truncate font-mono text-xs text-ink" title={page.path}>
                    {page.path}
                </h2>
                <div className="flex shrink-0 items-center gap-1">
                    <IconButton
                        icon={OpenInNewIcon}
                        label={copy.pages.openOnSite}
                        variant="ghost"
                        size="sm"
                        disabled={!reachable}
                        title={reachable ? copy.pages.openOnSite : copy.pages.summary.notOnSite}
                        onClick={() => {
                            void openExternal(url);
                        }}
                    />
                    <IconButton
                        data-page-open={true}
                        icon={EditNoteIcon}
                        label={copy.pages.summary.open}
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                            onOpen(page.id);
                        }}
                    />
                </div>
            </header>

            <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-auto p-3">
                <div className="flex flex-col gap-1.5">
                    <p className="text-sm font-semibold text-ink">
                        {page.title === "" ? copy.pages.untitled : page.title}
                    </p>
                    <div className="flex flex-wrap items-center gap-1.5">
                        <StatusBadge tone={statusTone(page.status)}>{page.status}</StatusBadge>
                        <StatusBadge tone="muted" dot={false}>
                            {page.wpType}
                        </StatusBadge>
                        {page.drift ? (
                            <StatusBadge tone="warn" icon={SyncProblemIcon}>
                                {copy.pages.drift.badge}
                            </StatusBadge>
                        ) : null}
                    </div>
                </div>

                <div className="flex flex-col gap-1.5">
                    <SectionLabel>{copy.pages.summary.entity}</SectionLabel>
                    {page.entityId === null ? (
                        <span className="flex items-center gap-1.5 text-sm text-ink-faint">
                            <LinkOffIcon size={16} className="shrink-0" />
                            {copy.pages.summary.noEntity}
                        </span>
                    ) : (
                        <div className="flex items-center gap-2">
                            <Icon size={16} className="shrink-0 text-accent" />
                            <Link
                                to={`/s/${siteId}/graph/${page.entityId}`}
                                className="min-w-0 flex-1 truncate text-sm"
                            >
                                {entity.data?.entity.name ?? copy.pages.mapped}
                            </Link>
                            {canonical ? (
                                <StatusBadge tone="ok" icon={VerifiedIcon}>
                                    {copy.pages.detail.canonical}
                                </StatusBadge>
                            ) : null}
                        </div>
                    )}
                </div>

                <div className="flex flex-col gap-1.5">
                    <SectionLabel>{copy.pages.summary.meta}</SectionLabel>
                    <dl className="flex flex-col gap-1.5">
                        <Row label={copy.pages.detail.metaTitle}>
                            {page.metaTitle === "" ? copy.pages.untitled : page.metaTitle}
                        </Row>
                        <Row label={copy.pages.detail.metaDescription}>
                            <span title={page.metaDescription}>
                                {copy.pages.detail.characters(page.metaDescription.length)}
                            </span>
                        </Row>
                        <Row label={copy.pages.detail.canonicalUrl}>
                            <span className="font-mono text-xs">{page.canonical}</span>
                        </Row>
                        <Row label={copy.pages.columns.synced}>
                            <span title={absoluteTime(page.lastSyncedAt)}>{relativeTime(page.lastSyncedAt)}</span>
                        </Row>
                    </dl>
                </div>

                <div className="flex flex-col gap-1.5">
                    <div className="flex items-baseline justify-between gap-2">
                        <SectionLabel>{copy.pages.summary.links}</SectionLabel>
                        <span className="font-mono text-2xs text-ink-faint">
                            {copy.pages.detail.linksSummary(links.length, generated)}
                        </span>
                    </div>
                    {links.length === 0 ? (
                        <p className="text-xs text-ink-dim">{copy.pages.summary.noLinks}</p>
                    ) : (
                        <ul className="flex flex-col gap-1">
                            {links.slice(0, shownLinks).map((held) => (
                                <li key={held.id} className="flex min-w-0 items-center gap-1.5">
                                    <LinkIcon size={12} className="shrink-0 text-ink-faint" />
                                    <span className="truncate font-mono text-2xs text-ink-soft" title={held.toUrl}>
                                        {held.toUrl}
                                    </span>
                                </li>
                            ))}
                            {links.length > shownLinks ? (
                                <li className="text-2xs text-ink-faint">
                                    {copy.pages.summary.moreLinks(links.length - shownLinks)}
                                </li>
                            ) : null}
                        </ul>
                    )}
                </div>

                <div className="flex flex-col gap-1.5">
                    <SectionLabel>{copy.pages.summary.report}</SectionLabel>
                    {report.data === undefined ? (
                        <p className="text-xs text-ink-dim">{copy.pages.detail.noReport}</p>
                    ) : (
                        <Link
                            to={`/s/${siteId}/runs/${report.data.runId}/items/${report.data.itemId}`}
                            className="flex items-center gap-2 rounded-md border border-hairline bg-inset px-2.5 py-2 hover:bg-raised"
                        >
                            <TaskAltIcon size={16} className="shrink-0" />
                            <span
                                className="min-w-0 flex-1 truncate font-mono text-2xs text-ink-faint"
                                title={absoluteTime(report.data.finishedAt)}
                            >
                                {relativeTime(report.data.finishedAt)}
                            </span>
                            <ChevronRightIcon size={16} className="shrink-0 text-ink-dim" />
                        </Link>
                    )}
                </div>

                <Link to={`/s/${siteId}/pages/${page.id}${search}`} className="mt-auto">
                    <Button variant="primary" icon={EditNoteIcon} className="w-full">
                        {copy.pages.summary.open}
                    </Button>
                </Link>
            </div>
        </div>
    );
}
