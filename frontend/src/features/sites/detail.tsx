import type { ReactElement, ReactNode } from "react";
import { useState } from "react";
import { Link } from "react-router";

import { copy } from "../../copy/index.js";
import { react } from "../../data/errors.js";
import { useTestConnection } from "../../data/hooks/sites.js";
import { usePolicy, useTemplate } from "../../data/hooks/templates.js";
import { isBrowsable, openExternal } from "../../data/host.js";
import type { Reachability, Site } from "../../data/types.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import {
    Banner,
    Button,
    DeleteIcon,
    EditNoteIcon,
    IconButton,
    OpenInNewIcon,
    SectionLabel,
    SpaceDashboardIcon,
    StatusBadge,
    TravelExploreIcon,
} from "../../ui/index.js";
import { PluginPanel } from "./plugin-panel.js";
import { ReachabilityReport } from "./reachability.js";
import { siteStatusTone } from "./status.js";

function Row({ label, children }: { label: string; children: ReactNode }): ReactElement {
    return (
        <div className="flex items-baseline justify-between gap-3">
            <dt className="shrink-0 text-2xs font-semibold tracking-label text-ink-faint uppercase">{label}</dt>
            <dd className="min-w-0 truncate text-sm text-ink">{children}</dd>
        </div>
    );
}

function Defaults({ site }: { site: Site }): ReactElement {
    const template = useTemplate(site.defaults.templateId ?? null);
    const policy = usePolicy(site.defaults.linkPolicyId ?? null);
    const roles = Object.keys(site.defaults.modelProfiles ?? {}).length;
    const said = copy.sites.defaults;

    return (
        <div className="flex flex-col gap-1.5">
            <SectionLabel>{said.title}</SectionLabel>
            <dl className="flex flex-col gap-1.5">
                <Row label={said.template}>
                    <Link to={`/s/${site.id}/templates`}>{template.data?.template.name ?? said.none}</Link>
                </Row>
                <Row label={said.linkPolicy}>
                    <Link to={`/s/${site.id}/templates`}>{policy.data?.policy.name ?? said.none}</Link>
                </Row>
                <Row label={said.modelRoles}>
                    <Link to="/settings/models">{roles === 0 ? said.none : said.roles(roles)}</Link>
                </Row>
            </dl>
        </div>
    );
}

export interface SiteDetailProps {
    site: Site;
    onEdit: () => void;
    onDelete: () => void;
}

export function SiteDetail({ site, onEdit, onDelete }: SiteDetailProps): ReactElement {
    const probe = useTestConnection();
    const [reachability, setReachability] = useState<Reachability | null>(null);
    const [problem, setProblem] = useState<string | null>(null);

    const runTest = (): void => {
        setReachability(null);
        setProblem(null);
        probe.mutate(
            { siteId: site.id },
            {
                onSuccess: (answered) => {
                    setReachability(answered.reachability);
                },
                onError: (thrown) => {
                    const reaction = react(thrown);
                    setProblem(reaction.kind === "field" || reaction.kind === "form" ? reaction.message : null);
                },
            },
        );
    };

    return (
        <div className="flex h-full min-h-0 flex-col">
            <header className="flex h-8 shrink-0 items-center justify-between gap-3 border-b border-hairline px-3">
                <h2 className="truncate text-xs font-semibold text-ink">{site.name}</h2>
                <div className="flex shrink-0 items-center gap-1">
                    <IconButton
                        data-site-edit={true}
                        icon={EditNoteIcon}
                        label={copy.sites.edit}
                        variant="ghost"
                        size="sm"
                        onClick={onEdit}
                    />
                    <IconButton
                        icon={DeleteIcon}
                        label={copy.sites.deleteConfirm}
                        variant="ghost"
                        size="sm"
                        onClick={onDelete}
                    />
                </div>
            </header>
            <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-auto p-3">
                <dl className="flex flex-col gap-1.5">
                    <Row label={copy.sites.columns.baseUrl}>
                        {isBrowsable(site.baseUrl) ? (
                            <button
                                type="button"
                                title={copy.app.openExternal}
                                onClick={() => {
                                    void openExternal(site.baseUrl);
                                }}
                                className="flex min-w-0 items-center gap-1 font-mono text-xs text-accent hover:underline"
                            >
                                <span className="truncate">{site.baseUrl}</span>
                                <OpenInNewIcon size={12} className="shrink-0" />
                            </button>
                        ) : (
                            <span className="font-mono text-xs">{site.baseUrl}</span>
                        )}
                    </Row>
                    <Row label={copy.sites.field.username}>
                        <span className="font-mono text-xs">{site.username}</span>
                    </Row>
                    <Row label={copy.sites.columns.status}>
                        <StatusBadge tone={siteStatusTone(site.status)}>{site.status}</StatusBadge>
                    </Row>
                    <Row label={copy.sites.columns.created}>
                        <span title={absoluteTime(site.createdAt)}>{relativeTime(site.createdAt)}</span>
                    </Row>
                </dl>

                {site.allowInsecure ? <Banner tone="warn" title={copy.sites.insecureWarning} /> : null}

                <PluginPanel site={site} />

                <Defaults site={site} />

                <div className="flex flex-col gap-2">
                    <Button variant="secondary" icon={TravelExploreIcon} busy={probe.isPending} onClick={runTest}>
                        {probe.isPending ? copy.sites.testing : copy.sites.test}
                    </Button>
                    {problem === null ? null : <Banner tone="danger" title={problem} />}
                    {reachability === null ? null : <ReachabilityReport result={reachability} />}
                </div>

                <Link to={`/s/${site.id}/overview`} className="mt-auto">
                    <Button variant="primary" icon={SpaceDashboardIcon} className="w-full">
                        {copy.sites.open}
                    </Button>
                </Link>
            </div>
        </div>
    );
}
