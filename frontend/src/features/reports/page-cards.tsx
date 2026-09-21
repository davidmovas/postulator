import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { PageReport } from "../../data/types.js";
import { isBrowsable, openExternal } from "../../data/host.js";
import { Button, OpenInNewIcon, Panel, PanelHeader, StatusBadge, cx } from "../../ui/index.js";
import type { Finding } from "../runs/artifacts.js";
import { judgeView, publishView, relinkView, validationView, weigh } from "../runs/artifacts.js";

interface FindingListProps {
    findings: readonly Finding[];
}

function FindingList({ findings }: FindingListProps): ReactElement {
    if (findings.length === 0) {
        return <p className="text-2xs text-ink-faint">{copy.reports.pages.noFindings}</p>;
    }
    return (
        <ul className="flex flex-col gap-1">
            {findings.map((finding, at) => (
                <li key={`${finding.code}-${at}`} className="flex items-baseline gap-2">
                    <span
                        className={cx(
                            "shrink-0 font-mono text-2xs",
                            finding.severity === "error"
                                ? "text-danger"
                                : finding.severity === "warn"
                                  ? "text-warn"
                                  : "text-ink-faint",
                        )}
                    >
                        {finding.severity}
                    </span>
                    <span className="min-w-0 flex-1 text-xs text-ink-soft">
                        {finding.message === "" ? finding.code : finding.message}
                    </span>
                </li>
            ))}
        </ul>
    );
}

function Card({ title, badge, children }: { title: string; badge?: ReactElement; children: ReactElement }): ReactElement {
    return (
        <Panel>
            <PanelHeader title={title}>{badge}</PanelHeader>
            <div className="flex flex-col gap-2 p-3">{children}</div>
        </Panel>
    );
}

export interface PageReportCardsProps {
    report: PageReport;
}

export function PageReportCards({ report }: PageReportCardsProps): ReactElement {
    const validation = validationView(report.validation ?? null);
    const judge = judgeView(report.judge ?? null);
    const publish = publishView(report.publish ?? null);
    const relink = relinkView(report.relink ?? null);
    const findings = validation === null ? [] : [...validation.compliance, ...validation.structure];
    const counted = weigh(findings);

    return (
        <div className="flex flex-col gap-3">
            {validation === null ? null : (
                <Card
                    title={copy.reports.pages.validation}
                    badge={
                        <StatusBadge tone={counted.errors > 0 ? "danger" : counted.warnings > 0 ? "warn" : "ok"} dot={false}>
                            {copy.reports.pages.findings(findings.length)}
                        </StatusBadge>
                    }
                >
                    <>
                        <p className="flex gap-3 font-mono text-2xs text-ink-dim">
                            {validation.complianceScore === null ? null : (
                                <span>{copy.reports.pages.complianceScore(validation.complianceScore)}</span>
                            )}
                            {validation.structureScore === null ? null : (
                                <span>{copy.reports.pages.structureScore(validation.structureScore)}</span>
                            )}
                        </p>
                        <FindingList findings={findings} />
                    </>
                </Card>
            )}
            {judge === null ? null : (
                <Card
                    title={copy.reports.pages.judge}
                    badge={
                        <StatusBadge tone={judge.score >= 7 ? "ok" : judge.score >= 5 ? "warn" : "danger"} dot={false}>
                            {copy.reports.pages.score(judge.score)}
                        </StatusBadge>
                    }
                >
                    <>
                        {judge.issues.length === 0 ? null : (
                            <div className="flex flex-col gap-1">
                                <span className="text-2xs tracking-label text-ink-faint uppercase">
                                    {copy.reports.pages.issues}
                                </span>
                                <ul className="flex flex-col gap-0.5">
                                    {judge.issues.map((issue, at) => (
                                        <li key={at} className="text-xs text-ink-soft">
                                            {issue}
                                        </li>
                                    ))}
                                </ul>
                            </div>
                        )}
                        {judge.suggestions.length === 0 ? null : (
                            <div className="flex flex-col gap-1">
                                <span className="text-2xs tracking-label text-ink-faint uppercase">
                                    {copy.reports.pages.suggestions}
                                </span>
                                <ul className="flex flex-col gap-0.5">
                                    {judge.suggestions.map((suggestion, at) => (
                                        <li key={at} className="text-xs text-ink-soft">
                                            {suggestion}
                                        </li>
                                    ))}
                                </ul>
                            </div>
                        )}
                        {judge.issues.length === 0 && judge.suggestions.length === 0 ? (
                            <p className="text-2xs text-ink-faint">{copy.reports.pages.noFindings}</p>
                        ) : null}
                    </>
                </Card>
            )}
            {publish === null ? null : (
                <Card
                    title={copy.reports.pages.publish}
                    badge={
                        <StatusBadge tone={publish.status === "" ? "muted" : "ok"} dot={false}>
                            {publish.status === "" ? copy.reports.pages.notPublished : publish.status}
                        </StatusBadge>
                    }
                >
                    <>
                        {publish.url === "" ? null : (
                            <div className="flex min-w-0 flex-col gap-0.5">
                                <span className="text-2xs tracking-label text-ink-faint uppercase">
                                    {copy.reports.pages.liveUrl}
                                </span>
                                <span className="truncate font-mono text-xs text-ink-soft" title={publish.url}>
                                    {publish.url}
                                </span>
                            </div>
                        )}
                        <p className="flex flex-wrap gap-3 text-2xs text-ink-dim">
                            <span>
                                {copy.reports.pages.seoApplied}
                                <span className="ml-1 font-mono text-ink">{publish.seoApplied.length}</span>
                            </span>
                            <span>
                                {copy.reports.pages.seoSkipped}
                                <span className="ml-1 font-mono text-ink">{publish.skipped.length}</span>
                            </span>
                        </p>
                        <FindingList findings={publish.findings} />
                        <Button
                            size="sm"
                            variant="secondary"
                            icon={OpenInNewIcon}
                            disabled={!isBrowsable(publish.url)}
                            title={copy.app.openExternal}
                            onClick={() => {
                                void openExternal(publish.url);
                            }}
                        >
                            {copy.app.openExternal}
                        </Button>
                    </>
                </Card>
            )}
            {relink === null ? null : (
                <Card title={copy.reports.pages.relink}>
                    <>
                        <p className="flex flex-wrap gap-3 text-2xs text-ink-dim">
                            <span>
                                <span className="mr-1 font-mono text-ink">{relink.linked ?? 0}</span>
                                {copy.reports.pages.linked}
                            </span>
                            <span>
                                <span className="mr-1 font-mono text-ink">{relink.conflicts ?? 0}</span>
                                {copy.reports.pages.conflicts}
                            </span>
                            <span>
                                <span className="mr-1 font-mono text-ink">{relink.skipped ?? 0}</span>
                                {copy.reports.pages.skipped}
                            </span>
                        </p>
                        <ul className="flex flex-col gap-0.5">
                            {relink.neighbours.map((neighbour) => (
                                <li key={neighbour.pageId} className="flex items-baseline gap-2">
                                    <span className="min-w-0 flex-1 truncate font-mono text-xs text-ink-soft">
                                        {neighbour.path}
                                    </span>
                                    <span className="shrink-0 text-2xs text-ink-faint">{neighbour.outcome}</span>
                                </li>
                            ))}
                        </ul>
                    </>
                </Card>
            )}
        </div>
    );
}
