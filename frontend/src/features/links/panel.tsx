import type { ReactElement } from "react";
import { useMemo } from "react";
import { Link } from "react-router";

import { copy } from "../../copy/index.js";
import { useLinkAuditPage } from "../../data/hooks/reports.js";
import type { ExtraLink, PageAudit, RequiredLink } from "../../data/types.js";
import {
    Banner,
    Button,
    CheckCircleIcon,
    CloseIcon,
    cx,
    ErrorIcon,
    HourglassTopIcon,
    IconButton,
    LinkOffIcon,
    PlayCircleIcon,
    SectionLabel,
    SkeletonRows,
    StatusBadge,
    toneClasses,
} from "../../ui/index.js";
import type { IconComponent, Tone } from "../../ui/index.js";
import { pageStatusLabel, statusTone } from "../pages/labels.js";
import { classLabel, classTone, relationIcon } from "./labels.js";
import { groups } from "./model/detail.js";

interface RequiredRowProps {
    siteId: string;
    link: RequiredLink;
}

function stateTone(link: RequiredLink): Tone {
    switch (link.state) {
        case "placed":
            return "ok";
        case "target_unpublished":
            return "warn";
        case "awaiting_page":
        case "awaiting_target":
            return "info";
        default:
            return link.required ? "danger" : "warn";
    }
}

function stateIcon(link: RequiredLink): IconComponent {
    switch (link.state) {
        case "placed":
            return CheckCircleIcon;
        case "awaiting_page":
        case "awaiting_target":
            return HourglassTopIcon;
        default:
            return ErrorIcon;
    }
}

function RequiredRow({ siteId, link }: RequiredRowProps): ReactElement {
    const Icon = relationIcon(link.relation);
    return (
        <li className="flex flex-col gap-0.5 rounded-md px-1 py-1 hover:bg-inset" title={copy.links.stateHints[link.state]}>
            <span className="flex items-center gap-2 text-xs">
                <Icon size={14} className="shrink-0 text-ink-faint" />
                <Link to={`/s/${siteId}/graph/${link.targetEntityId}`} className="min-w-0 flex-1 truncate text-ink-soft hover:text-ink">
                    {link.targetEntityName}
                </Link>
                {link.required ? <span className="text-2xs text-ink-faint">{copy.links.panel.required}</span> : null}
                <StatusBadge tone={stateTone(link)} icon={stateIcon(link)} dot={false}>
                    {copy.links.states[link.state] ?? (link.satisfied ? copy.links.panel.satisfied : copy.links.panel.missing)}
                </StatusBadge>
            </span>
            <span className="flex items-center gap-2 pl-6 font-mono text-2xs text-ink-faint">
                <span className="truncate">{link.targetPath}</span>
                {link.satisfied ? (
                    <span className={cx("truncate", link.anchorAllowed ? "text-ink-dim" : "text-warn")} title={copy.links.panel.allowed(link.anchorsAllowed ?? [])}>
                        {copy.links.panel.anchor(link.anchor)}
                        {link.anchorAllowed ? "" : ` · ${copy.links.panel.anchorNotAllowed}`}
                    </span>
                ) : (
                    <span className="truncate text-ink-faint" title={copy.links.panel.allowed(link.anchorsAllowed ?? [])}>
                        {copy.links.panel.allowed(link.anchorsAllowed ?? [])}
                    </span>
                )}
            </span>
        </li>
    );
}

interface BlockedRowProps {
    siteId: string;
    link: RequiredLink;
}

function BlockedRow({ siteId, link }: BlockedRowProps): ReactElement {
    const Icon = relationIcon(link.relation);
    return (
        <li className="flex items-center gap-2 rounded-md px-1 py-1 text-xs hover:bg-inset">
            <Icon size={14} className="shrink-0 text-ink-faint" />
            <span className="min-w-0 flex-1 truncate text-ink-soft">{link.targetEntityName}</span>
            <LinkOffIcon size={12} className="shrink-0 text-danger" />
            <Link to={`/s/${siteId}/graph/${link.targetEntityId}`} className="shrink-0 text-2xs">
                {copy.links.panel.givePage}
            </Link>
        </li>
    );
}

interface ExtraRowProps {
    siteId: string;
    link: ExtraLink;
}

function ExtraRow({ siteId, link }: ExtraRowProps): ReactElement {
    return (
        <li className="flex flex-col gap-0.5 rounded-md px-1 py-1 text-xs hover:bg-inset">
            {link.toPageId === "" ? (
                <span className="truncate font-mono text-ink-soft" title={link.toUrl}>
                    {link.toUrl}
                </span>
            ) : (
                <Link to={`/s/${siteId}/pages/${link.toPageId}`} className="truncate font-mono text-ink-soft hover:text-ink" title={link.toUrl}>
                    {link.toUrl}
                </Link>
            )}
            <span className="font-mono text-2xs text-ink-faint">
                {copy.links.panel.anchor(link.anchor)} · {link.origin}
            </span>
        </li>
    );
}

interface SectionProps {
    title: string;
    count: number;
    children: ReactElement | null;
}

function Section({ title, count, children }: SectionProps): ReactElement {
    return (
        <section className="flex flex-col gap-1">
            <div className="flex items-baseline justify-between">
                <SectionLabel>{title}</SectionLabel>
                <span className="font-mono text-2xs text-ink-faint">{count}</span>
            </div>
            {count === 0 ? <p className="text-2xs text-ink-faint">{copy.links.panel.none}</p> : children}
        </section>
    );
}

export interface AuditPanelProps {
    siteId: string;
    row: PageAudit;
    onClose: () => void;
    onRelink: (pageId: string) => void;
}

export function AuditPanel({ siteId, row, onClose, onRelink }: AuditPanelProps): ReactElement {
    const detail = useLinkAuditPage(row.pageId);
    const grouped = useMemo(() => (detail.data === undefined ? null : groups(detail.data)), [detail.data]);
    const rules = detail.data?.rules ?? null;
    const skipped = row.skipReason !== "";
    const unwritten = !skipped && !row.onSite;

    return (
        <section aria-label={copy.links.panel.title} className="flex h-full min-h-0 flex-col bg-panel">
            <header className="flex items-start gap-2 border-b border-hairline p-3">
                <div className="min-w-0 flex-1">
                    <Link to={`/s/${siteId}/pages/${row.pageId}`} className="block truncate font-mono text-sm text-ink hover:text-accent" title={row.path}>
                        {row.path}
                    </Link>
                    <p className="flex flex-wrap items-center gap-2 text-2xs text-ink-dim">
                        <StatusBadge tone={statusTone(row.status)}>{pageStatusLabel(row.status)}</StatusBadge>
                        {row.entityName === "" ? null : (
                            <Link to={`/s/${siteId}/graph/${row.entityId}`} className="truncate">
                                {row.entityName}
                            </Link>
                        )}
                        <span>{row.orphan ? copy.links.panel.orphan : copy.links.panel.inbound(row.inbound)}</span>
                    </p>
                </div>
                <IconButton icon={CloseIcon} label={copy.links.panel.close} variant="ghost" size="sm" onClick={onClose} />
            </header>

            <div className="flex flex-col gap-4 p-3">
                {unwritten ? (
                    <Banner
                        tone="info"
                        icon={HourglassTopIcon}
                        title={row.status === "planned" ? copy.links.panel.notWritten : copy.links.panel.notOnSite}
                        body={row.status === "planned" ? copy.links.panel.notWrittenBody : copy.links.panel.notOnSiteBody}
                    />
                ) : null}
                {skipped ? (
                    <Banner tone="muted" title={copy.links.cell.skipped[row.skipReason] ?? row.skipReason} body={copy.links.panel.skipped[row.skipReason] ?? ""} />
                ) : detail.isPending || grouped === null ? (
                    <SkeletonRows rows={6} label={copy.links.loading} />
                ) : (
                    <>
                        <ul className="flex flex-wrap gap-x-3 gap-y-1 text-2xs text-ink-dim">
                            <li>
                                <span className={cx("font-mono", toneClasses.ok.ink)}>{grouped.counts.satisfied}</span> {copy.links.panel.satisfied}
                            </li>
                            {grouped.counts.missing > 0 ? (
                                <li>
                                    <span className={cx("font-mono", grouped.counts.missingRequired > 0 ? toneClasses.danger.ink : toneClasses.warn.ink)}>
                                        {grouped.counts.missing}
                                    </span>{" "}
                                    {copy.links.panel.missing}
                                </li>
                            ) : null}
                            {grouped.counts.unpublished > 0 ? (
                                <li>
                                    <span className={cx("font-mono", toneClasses.warn.ink)}>{grouped.counts.unpublished}</span> {copy.links.panel.unpublished}
                                </li>
                            ) : null}
                            {grouped.counts.pending > 0 ? (
                                <li>
                                    <span className={cx("font-mono", toneClasses.info.ink)}>{grouped.counts.pending}</span> {copy.links.panel.waiting}
                                </li>
                            ) : null}
                            {grouped.counts.blocked > 0 ? (
                                <li>
                                    <span className={cx("font-mono", toneClasses.danger.ink)}>{grouped.counts.blocked}</span> {copy.links.panel.blocked.toLowerCase()}
                                </li>
                            ) : null}
                            {grouped.counts.anchorNotAllowed > 0 ? (
                                <li>
                                    <span className={cx("font-mono", toneClasses.warn.ink)}>{grouped.counts.anchorNotAllowed}</span> {copy.links.panel.anchorNotAllowed}
                                </li>
                            ) : null}
                        </ul>

                        <Section title={copy.links.panel.up} count={grouped.up.length}>
                            <ul>
                                {grouped.up.map((link) => (
                                    <RequiredRow key={`${link.relation}:${link.targetEntityId}`} siteId={siteId} link={link} />
                                ))}
                            </ul>
                        </Section>
                        <Section title={copy.links.panel.down} count={grouped.down.length}>
                            <ul>
                                {grouped.down.map((link) => (
                                    <RequiredRow key={`${link.relation}:${link.targetEntityId}`} siteId={siteId} link={link} />
                                ))}
                            </ul>
                        </Section>
                        <Section title={copy.links.panel.sibling} count={grouped.sibling.length}>
                            <ul>
                                {grouped.sibling.map((link) => (
                                    <RequiredRow key={`${link.relation}:${link.targetEntityId}`} siteId={siteId} link={link} />
                                ))}
                            </ul>
                        </Section>
                        {grouped.blocked.length === 0 ? null : (
                            <section className="flex flex-col gap-1">
                                <div className="flex items-baseline justify-between">
                                    <SectionLabel>{copy.links.panel.blocked}</SectionLabel>
                                    <span className="font-mono text-2xs text-ink-faint">{grouped.blocked.length}</span>
                                </div>
                                <p className="text-2xs text-ink-faint">{copy.links.panel.blockedHint}</p>
                                <ul>
                                    {grouped.blocked.map((link) => (
                                        <BlockedRow key={`${link.relation}:${link.targetEntityId}`} siteId={siteId} link={link} />
                                    ))}
                                </ul>
                            </section>
                        )}
                        <Section title={copy.links.panel.extra} count={grouped.counts.offGraph}>
                            <div className="flex flex-col gap-2">
                                {grouped.extra.map((group) => (
                                    <div key={group.kind} className="flex flex-col gap-0.5">
                                        <StatusBadge tone={classTone(group.kind)} dot={false} className="self-start">
                                            {classLabel(group.kind)} · {group.links.length}
                                        </StatusBadge>
                                        <ul>
                                            {group.links.map((link, position) => (
                                                <ExtraRow key={`${link.toUrl}:${position}`} siteId={siteId} link={link} />
                                            ))}
                                        </ul>
                                    </div>
                                ))}
                            </div>
                        </Section>
                        {rules === null ? null : (
                            <p className="font-mono text-2xs text-ink-faint">{copy.links.panel.rules(rules.upDepth, rules.downLinks, rules.siblingMinWeight)}</p>
                        )}
                    </>
                )}
                <footer className="flex justify-end border-t border-hairline pt-3">
                    <Button
                        size="sm"
                        variant="secondary"
                        icon={PlayCircleIcon}
                        disabled={skipped || unwritten}
                        onClick={() => {
                            onRelink(row.pageId);
                        }}
                    >
                        {copy.links.relink.page}
                    </Button>
                </footer>
            </div>
        </section>
    );
}
