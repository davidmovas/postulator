import type { ReactElement } from "react";
import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router";

import { copy } from "../../copy/index.js";
import { flatten } from "../../data/call.js";
import { useSites } from "../../data/hooks/sites.js";
import type { SiteSort } from "../../data/sorts.js";
import type { Site, SiteFilter } from "../../data/types.js";
import { siteStatuses } from "../../generated/vocab.js";
import {
    AddIcon,
    Button,
    CountBadge,
    EmptyState,
    Input,
    Screen,
    Segmented,
    Toolbar,
    TravelExploreIcon,
} from "../../ui/index.js";
import type { SegmentedOption } from "../../ui/index.js";
import { SiteDeleteDialog } from "./delete-dialog.js";
import { SiteDetail } from "./detail.js";
import { SiteForm } from "./site-form.js";
import { SiteTable } from "./table.js";

const anyStatus = "any";

const statusOptions: readonly SegmentedOption<string>[] = [
    { value: anyStatus, label: copy.sites.anyStatus },
    ...siteStatuses.map((held) => ({ value: held, label: held })),
];

type FormState = { mode: "create" } | { mode: "edit"; site: Site } | null;

function matching(rows: readonly Site[], search: string): readonly Site[] {
    const needle = search.trim().toLowerCase();
    if (needle === "") {
        return rows;
    }
    return rows.filter(
        (row) => row.name.toLowerCase().includes(needle) || row.baseUrl.toLowerCase().includes(needle),
    );
}

export function SitesScreen(): ReactElement {
    const navigate = useNavigate();
    const [searchParams, setSearchParams] = useSearchParams();
    const [status, setStatus] = useState<string>(anyStatus);
    const [search, setSearch] = useState("");
    const [sort, setSort] = useState<SiteSort | null>(null);
    const [selectedId, setSelectedId] = useState<string | null>(null);
    const [form, setForm] = useState<FormState>(null);
    const [deleting, setDeleting] = useState<Site | null>(null);

    const filter = useMemo<SiteFilter>(() => (status === anyStatus ? {} : { status }), [status]);
    const sites = useSites(filter, sort);
    const rows = flatten(sites.data?.pages);
    const shown = matching(rows, search);

    const ids = shown.map((row) => row.id).join("|");

    useEffect(() => {
        const listed = ids === "" ? [] : ids.split("|");
        setSelectedId((held) => {
            if (listed.length === 0) {
                return null;
            }
            return held !== null && listed.includes(held) ? held : listed[0];
        });
    }, [ids]);

    const asked = searchParams.get("action");

    useEffect(() => {
        if (asked !== "new") {
            return;
        }
        setForm({ mode: "create" });
        setSearchParams({}, { replace: true });
    }, [asked, setSearchParams]);

    const selected = shown.find((row) => row.id === selectedId) ?? null;
    const nothingAtAll = !sites.isPending && rows.length === 0 && status === anyStatus && search === "";

    const add = (
        <Button
            variant="primary"
            icon={AddIcon}
            onClick={() => {
                setForm({ mode: "create" });
            }}
        >
            {copy.sites.add}
        </Button>
    );

    const overlays = (
        <>
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
        </>
    );

    if (nothingAtAll) {
        return (
            <Screen title={copy.sites.title} actions={add}>
                <div className="flex h-full items-center justify-center">
                    <EmptyState
                        icon={TravelExploreIcon}
                        title={copy.sites.emptyTitle}
                        body={copy.sites.emptyBody}
                        actions={add}
                        className="w-80"
                    />
                </div>
                {overlays}
            </Screen>
        );
    }

    return (
        <Screen
            title={copy.sites.title}
            badge={<CountBadge tone="muted" count={rows.length} />}
            actions={add}
            variant="split"
            right={
                selected === null ? null : (
                    <SiteDetail
                        site={selected}
                        onEdit={() => {
                            setForm({ mode: "edit", site: selected });
                        }}
                        onDelete={() => {
                            setDeleting(selected);
                        }}
                    />
                )
            }
            toolbar={
                <Toolbar label={copy.sites.title}>
                    <Segmented
                        label={copy.sites.columns.status}
                        value={status}
                        options={statusOptions}
                        onValueChange={setStatus}
                    />
                    <div className="w-56">
                        <Input
                            type="search"
                            aria-label={copy.sites.search}
                            placeholder={copy.sites.search}
                            value={search}
                            onChange={(event) => {
                                setSearch(event.target.value);
                            }}
                        />
                    </div>
                </Toolbar>
            }
        >
            <div className="min-h-0 flex-1 overflow-auto">
                {shown.length === 0 && !sites.isPending ? (
                    <p className="p-4 text-xs text-ink-dim">{copy.sites.noMatches}</p>
                ) : (
                    <SiteTable
                        rows={shown}
                        loading={sites.isPending}
                        selectedId={selectedId}
                        sort={sort}
                        onSortChange={setSort}
                        onSelect={setSelectedId}
                        onEnter={(id) => {
                            void navigate(`/s/${id}/overview`);
                        }}
                    />
                )}
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
            {overlays}
        </Screen>
    );
}
