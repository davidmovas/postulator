import type { ReactElement } from "react";
import { useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router";

import { readLastSite } from "../../../app/site-memory.js";
import { copy } from "../../../copy/index.js";
import { flatten } from "../../../data/call.js";
import { failure } from "../../../data/errors.js";
import { useSites } from "../../../data/hooks/sites.js";
import { usePageTypes, useTemplates } from "../../../data/hooks/templates.js";
import type { PageType } from "../../../data/types.js";
import {
    AddIcon,
    Banner,
    Button,
    CategoryIcon,
    DenseTable,
    EmptyState,
    Panel,
    PanelHeader,
    PublicIcon,
    Select,
    SkeletonRows,
    StatusBadge,
    TableCell,
    TableHead,
    TableRow,
    toneClasses,
} from "../../../ui/index.js";
import { presetIcon, presetLabel, presetTone } from "./labels.js";
import { orderedTypes, pickSite } from "./model.js";
import type { TemplateChoice } from "./type-dialogs.js";
import { TypeFormDialog } from "./type-dialogs.js";
import { TypeDrawer } from "./type-drawer.js";

const said = copy.pageTypes.screen;

const columns = "minmax(0,1.6fr) minmax(0,1fr) 6rem minmax(0,1.2fr)";

const templatePage = 100;

interface SiteTypesProps {
    siteId: string;
    typeId: string | null;
    onOpen: (typeId: string | null) => void;
}

function SiteTypes({ siteId, typeId, onOpen }: SiteTypesProps): ReactElement {
    const listed = usePageTypes(siteId);
    const globals = useTemplates({ scope: "global" }, null, templatePage);
    const locals = useTemplates({ siteId }, null, templatePage);
    const [creating, setCreating] = useState(false);

    const templates: TemplateChoice[] = useMemo(
        () =>
            [...flatten(globals.data?.pages), ...flatten(locals.data?.pages)].map((held) => ({ id: held.id, name: held.name })),
        [globals.data, locals.data],
    );
    const names = useMemo(() => new Map(templates.map((held) => [held.id, held.name] as const)), [templates]);
    const types = useMemo(() => orderedTypes(listed.data?.types ?? []), [listed.data]);
    const open = types.find((held) => held.id === typeId) ?? null;

    const templateOf = (type: PageType): string =>
        type.templateId === null ? said.siteDefault : (names.get(type.templateId) ?? said.siteDefault);

    return (
        <Panel>
            <PanelHeader title={said.title}>
                <Button
                    size="sm"
                    variant="primary"
                    icon={AddIcon}
                    data-page-type-new={true}
                    onClick={() => {
                        setCreating(true);
                    }}
                >
                    {said.create}
                </Button>
            </PanelHeader>
            {listed.isPending ? (
                <SkeletonRows rows={6} label={said.loading} className="p-3" />
            ) : listed.error !== null ? (
                <div className="p-3">
                    <Banner
                        tone="danger"
                        title={failure(listed.error).message}
                        actions={
                            <Button
                                size="sm"
                                onClick={() => {
                                    void listed.refetch();
                                }}
                            >
                                {copy.app.retry}
                            </Button>
                        }
                    />
                </div>
            ) : types.length === 0 ? (
                <div className="p-3">
                    <EmptyState icon={CategoryIcon} title={said.empty} body={said.emptyBody} />
                </div>
            ) : (
                <DenseTable columns={columns} label={said.title}>
                    <TableHead>
                        <span>{said.columns.name}</span>
                        <span>{said.columns.preset}</span>
                        <span className="text-right">{said.columns.pages}</span>
                        <span>{said.columns.template}</span>
                    </TableHead>
                    {types.map((type) => {
                        const Icon = presetIcon(type.preset);
                        return (
                            <TableRow
                                key={type.id}
                                interactive={true}
                                selected={type.id === typeId}
                                data-page-type-row={type.preset}
                                aria-label={said.open(type.name)}
                                onClick={() => {
                                    onOpen(type.id);
                                }}
                            >
                                <TableCell title={type.name}>
                                    <span className="flex min-w-0 items-center gap-1.5">
                                        <Icon size={14} className={`shrink-0 ${toneClasses[presetTone(type.preset)].ink}`} />
                                        <span className="truncate">{type.name}</span>
                                    </span>
                                </TableCell>
                                <TableCell muted={true}>
                                    <span className="flex items-center gap-1.5">
                                        {presetLabel(type.preset)}
                                        {type.writable ? null : (
                                            <StatusBadge tone="muted" dot={false}>
                                                {said.readOnly}
                                            </StatusBadge>
                                        )}
                                    </span>
                                </TableCell>
                                <TableCell mono={true} align="right" muted={type.pages === 0}>
                                    {String(type.pages)}
                                </TableCell>
                                <TableCell muted={type.templateId === null} title={templateOf(type)}>
                                    {templateOf(type)}
                                </TableCell>
                            </TableRow>
                        );
                    })}
                </DenseTable>
            )}
            <TypeFormDialog
                open={creating}
                siteId={siteId}
                type={null}
                templates={templates}
                onOpenChange={setCreating}
                onCreated={(id) => {
                    onOpen(id);
                }}
            />
            {open === null ? null : (
                <TypeDrawer
                    key={open.id}
                    siteId={siteId}
                    type={open}
                    types={types}
                    catalog={listed.data?.fields ?? []}
                    templates={templates}
                    templateName={templateOf(open)}
                    onClose={() => {
                        onOpen(null);
                    }}
                />
            )}
        </Panel>
    );
}

export function PageTypesScreen(): ReactElement {
    const [searchParams, setSearchParams] = useSearchParams();
    const sites = useSites();
    const rows = flatten(sites.data?.pages);
    const siteId = pickSite(searchParams.get("site"), readLastSite(), rows.map((site) => site.id));
    const typeId = searchParams.get("type");

    const go = (site: string, type: string | null): void => {
        const next = new URLSearchParams({ site });
        if (type !== null) {
            next.set("type", type);
        }
        setSearchParams(next, { replace: true });
    };

    return (
        <div className="mx-auto flex w-full max-w-4xl flex-col gap-4">
            <div className="flex flex-wrap items-end justify-between gap-3">
                <p className="max-w-xl text-sm text-ink-soft">{said.body}</p>
                {siteId === null ? null : (
                    <div className="w-64">
                        <Select
                            aria-label={said.site}
                            value={siteId}
                            options={rows.map((site) => ({ value: site.id, label: site.name }))}
                            onValueChange={(site) => {
                                go(site, null);
                            }}
                        />
                    </div>
                )}
            </div>
            {sites.isPending ? (
                <Panel>
                    <SkeletonRows rows={4} label={said.loading} className="p-3" />
                </Panel>
            ) : siteId === null ? (
                <Panel>
                    <div className="p-3">
                        <EmptyState
                            icon={PublicIcon}
                            title={said.noSites}
                            actions={
                                <Link to="/sites">
                                    <Button size="sm">{said.openSites}</Button>
                                </Link>
                            }
                        />
                    </div>
                </Panel>
            ) : (
                <SiteTypes
                    key={siteId}
                    siteId={siteId}
                    typeId={typeId}
                    onOpen={(type) => {
                        go(siteId, type);
                    }}
                />
            )}
        </div>
    );
}
