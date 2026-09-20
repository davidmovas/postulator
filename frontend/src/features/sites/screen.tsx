import type { ReactElement } from "react";
import { useEffect, useMemo, useState } from "react";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { useSites } from "../../data/hooks/sites.js";
import { isBrowsable, openExternal } from "../../data/host.js";
import type { SiteSort } from "../../data/sorts.js";
import type { Site, SiteFilter } from "../../data/types.js";
import { absoluteTime, relativeTime } from "../../domain/format.js";
import { siteStatuses } from "../../generated/vocab.js";
import { ReadinessChecklist } from "../onboarding/index.js";
import {
    AddIcon,
    Button,
    DenseTable,
    EmptyState,
    ExtensionIcon,
    ExtensionOffIcon,
    IconButton,
    OpenInNewIcon,
    Panel,
    PanelHeader,
    Select,
    SkeletonRows,
    SortableHeader,
    StatusBadge,
    TableCell,
    TableHead,
    TableRow,
    TravelExploreIcon,
} from "../../ui/index.js";
import type { SelectOption } from "../../ui/index.js";
import { SiteDeleteDialog } from "./delete-dialog.js";
import { SiteDetail } from "./detail.js";
import { SiteForm } from "./site-form.js";
import { siteStatusTone } from "./status.js";

const anyStatus = "any";

const columns = "minmax(8rem,1.4fr) minmax(10rem,2fr) 5.5rem 7rem 6rem";

type FormState = { mode: "create" } | { mode: "edit"; site: Site } | null;

function nextSort(current: SiteSort | null, field: SiteSort["field"]): SiteSort {
    if (current !== null && current.field === field) {
        return { field, desc: !current.desc };
    }
    return { field, desc: field === "createdAt" };
}

export function SitesScreen(): ReactElement {
    const [status, setStatus] = useState<string>(anyStatus);
    const [sort, setSort] = useState<SiteSort | null>(null);
    const [selectedId, setSelectedId] = useState<string | null>(null);
    const [form, setForm] = useState<FormState>(null);
    const [deleting, setDeleting] = useState<Site | null>(null);

    const filter = useMemo<SiteFilter>(
        () => (status === anyStatus ? {} : { status }),
        [status],
    );

    const sites = useSites(filter, sort);
    const rows = flatten(sites.data?.pages);

    const ids = rows.map((row) => row.id).join("|");

    useEffect(() => {
        const listed = ids === "" ? [] : ids.split("|");
        setSelectedId((held) => {
            if (listed.length === 0) {
                return null;
            }
            return held !== null && listed.includes(held) ? held : listed[0];
        });
    }, [ids]);

    const selected = rows.find((row) => row.id === selectedId) ?? null;

    const statusOptions: SelectOption<string>[] = [
        { value: anyStatus, label: copy.sites.anyStatus },
        ...siteStatuses.map((held) => ({ value: held, label: held })),
    ];

    const empty = !sites.isLoading && rows.length === 0;

    return (
        <div className="flex h-full min-h-0 flex-col gap-3 p-4">
            <header className="flex items-start justify-between gap-4">
                <div className="flex min-w-0 flex-col gap-1">
                    <h1 className="text-xl font-semibold tracking-tight text-ink">{copy.sites.title}</h1>
                    <p className="max-w-2xl text-sm text-ink-soft">{copy.sites.subtitle}</p>
                </div>
                <div className="flex shrink-0 items-center gap-2">
                    <div className="w-32">
                        <Select
                            aria-label={copy.sites.columns.status}
                            value={status}
                            options={statusOptions}
                            onValueChange={setStatus}
                        />
                    </div>
                    <Button
                        variant="primary"
                        icon={AddIcon}
                        onClick={() => {
                            setForm({ mode: "create" });
                        }}
                    >
                        {copy.sites.add}
                    </Button>
                </div>
            </header>

            {empty ? (
                <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-auto">
                    <EmptyState
                        icon={TravelExploreIcon}
                        title={copy.sites.title}
                        body={copy.empty.sites}
                        actions={
                            <Button
                                variant="primary"
                                icon={AddIcon}
                                onClick={() => {
                                    setForm({ mode: "create" });
                                }}
                            >
                                {copy.sites.add}
                            </Button>
                        }
                    />
                    <ReadinessChecklist siteId={null} className="max-w-4xl" />
                </div>
            ) : (
                <div className="flex min-h-0 flex-1 gap-3">
                    <Panel className="min-w-0 flex-1">
                        <PanelHeader title={copy.sites.title} />
                        <div className="min-h-0 flex-1 overflow-auto">
                            <DenseTable columns={columns} label={copy.sites.title}>
                                <TableHead>
                                    <SortableHeader
                                        active={sort?.field === "name"}
                                        direction={sort?.desc === true ? "desc" : "asc"}
                                        onToggle={() => {
                                            setSort(nextSort(sort, "name"));
                                        }}
                                    >
                                        {copy.sites.columns.name}
                                    </SortableHeader>
                                    <span role="columnheader">{copy.sites.columns.baseUrl}</span>
                                    <span role="columnheader">{copy.sites.columns.status}</span>
                                    <span role="columnheader">{copy.sites.columns.plugin}</span>
                                    <SortableHeader
                                        active={sort?.field === "createdAt"}
                                        direction={sort?.desc === true ? "desc" : "asc"}
                                        align="right"
                                        onToggle={() => {
                                            setSort(nextSort(sort, "createdAt"));
                                        }}
                                    >
                                        {copy.sites.columns.created}
                                    </SortableHeader>
                                </TableHead>
                                {sites.isLoading ? (
                                    <SkeletonRows rows={6} label={copy.app.loading} className="p-3" />
                                ) : (
                                    rows.map((row) => (
                                        <TableRow
                                            key={row.id}
                                            interactive={true}
                                            selected={row.id === selectedId}
                                            tabIndex={0}
                                            onClick={() => {
                                                setSelectedId(row.id);
                                            }}
                                            onKeyDown={(event) => {
                                                if (event.key === "Enter" || event.key === " ") {
                                                    event.preventDefault();
                                                    setSelectedId(row.id);
                                                }
                                            }}
                                        >
                                            <TableCell>{row.name}</TableCell>
                                            <TableCell mono={true} muted={true} title={row.baseUrl}>
                                                <span className="flex min-w-0 items-center gap-1">
                                                    <span className="truncate">{row.baseUrl}</span>
                                                    {isBrowsable(row.baseUrl) ? (
                                                        <IconButton
                                                            icon={OpenInNewIcon}
                                                            label={copy.app.openExternal}
                                                            variant="ghost"
                                                            size="sm"
                                                            onClick={(event) => {
                                                                event.stopPropagation();
                                                                void openExternal(row.baseUrl);
                                                            }}
                                                        />
                                                    ) : null}
                                                </span>
                                            </TableCell>
                                            <TableCell>
                                                <StatusBadge tone={siteStatusTone(row.status)}>
                                                    {row.status}
                                                </StatusBadge>
                                            </TableCell>
                                            <TableCell>
                                                {row.plugin.installed ? (
                                                    <StatusBadge
                                                        tone="ok"
                                                        icon={ExtensionIcon}
                                                        dot={false}
                                                    >
                                                        {row.plugin.version === ""
                                                            ? copy.shell.pluginInstalled
                                                            : row.plugin.version}
                                                    </StatusBadge>
                                                ) : (
                                                    <StatusBadge
                                                        tone="warn"
                                                        icon={ExtensionOffIcon}
                                                        dot={false}
                                                    >
                                                        {copy.shell.pluginMissing}
                                                    </StatusBadge>
                                                )}
                                            </TableCell>
                                            <TableCell
                                                align="right"
                                                muted={true}
                                                title={absoluteTime(row.createdAt)}
                                            >
                                                {relativeTime(row.createdAt)}
                                            </TableCell>
                                        </TableRow>
                                    ))
                                )}
                            </DenseTable>
                            {sites.hasNextPage ? (
                                <div className="flex justify-center p-2">
                                    <Button
                                        size="sm"
                                        variant="secondary"
                                        busy={sites.isFetchingNextPage}
                                        onClick={() => {
                                            void sites.fetchNextPage();
                                        }}
                                    >
                                        {copy.app.loadMore}
                                    </Button>
                                </div>
                            ) : null}
                        </div>
                    </Panel>
                    {selected === null ? null : (
                        <SiteDetail
                            site={selected}
                            onEdit={() => {
                                setForm({ mode: "edit", site: selected });
                            }}
                            onDelete={() => {
                                setDeleting(selected);
                            }}
                        />
                    )}
                </div>
            )}

            {form === null ? null : (
                <SiteForm
                    key={form.mode === "edit" ? form.site.id : "new"}
                    site={form.mode === "edit" ? form.site : null}
                    onClose={() => {
                        setForm(null);
                    }}
                    onSaved={(saved) => {
                        setSelectedId(saved.id);
                    }}
                />
            )}

            {deleting === null ? null : (
                <SiteDeleteDialog
                    site={deleting}
                    onClose={() => {
                        setDeleting(null);
                    }}
                    onDeleted={() => {
                        setSelectedId(null);
                    }}
                />
            )}
        </div>
    );
}
