import type { QueryClient } from "@tanstack/react-query";
import { useQueryClient } from "@tanstack/react-query";
import type { ReactElement } from "react";
import { useLocation } from "react-router";

import { readLastSite } from "../../../app/site-memory.js";
import { copy } from "../../../copy/index.js";
import { useEntity } from "../../../data/hooks/graph.js";
import { usePage } from "../../../data/hooks/pages.js";
import { useRun } from "../../../data/hooks/runs.js";
import { useSchedule } from "../../../data/hooks/schedules.js";
import { useSite } from "../../../data/hooks/sites.js";
import { usePageTypes, useTemplate } from "../../../data/hooks/templates.js";
import { keys } from "../../../data/keys.js";
import type { PageType, PageTypeList } from "../../../data/types.js";
import { cx, Skeleton } from "../../../ui/index.js";
import type { Ref } from "./model/card.js";

const shortIdLength = 8;

interface ResolvedProps {
    label: string | null;
    pending: boolean;
    id: string;
    mono?: boolean;
}

function Resolved({ label, pending, id, mono = false }: ResolvedProps): ReactElement {
    if (pending && id !== "") {
        return <Skeleton width="6ch" height={12} className="inline-block align-middle" />;
    }
    if (label === null) {
        return (
            <span className="font-mono text-ink-faint" title={id}>
                {id === "" ? copy.agent.card.unknownRef : `${id.slice(0, shortIdLength)} · ${copy.agent.card.unknownRef}`}
            </span>
        );
    }
    return (
        <span className={cx("font-semibold text-ink", mono && "font-mono font-normal")} title={id}>
            {label}
        </span>
    );
}

function PageRef({ id }: { id: string }): ReactElement {
    const detail = usePage(id);
    return <Resolved label={detail.data?.page.path ?? null} pending={detail.isPending} id={id} mono={true} />;
}

function EntityRef({ id }: { id: string }): ReactElement {
    const detail = useEntity(id);
    return <Resolved label={detail.data?.entity.name ?? null} pending={detail.isPending} id={id} />;
}

function TemplateRef({ id }: { id: string }): ReactElement {
    const detail = useTemplate(id);
    return <Resolved label={detail.data?.template.name ?? null} pending={detail.isPending} id={id} />;
}

function SiteRef({ id }: { id: string }): ReactElement {
    const detail = useSite(id);
    return <Resolved label={detail.data?.site.name ?? null} pending={detail.isPending} id={id} />;
}

const sitePath = /^\/s\/([^/]+)/;

function cachedType(client: QueryClient, id: string): PageType | null {
    for (const [, held] of client.getQueriesData<PageTypeList>({ queryKey: keys.sites.typesAll() })) {
        const found = held?.types?.find((type) => type.id === id);
        if (found !== undefined) {
            return found;
        }
    }
    return null;
}

function PageTypeRef({ id }: { id: string }): ReactElement {
    const location = useLocation();
    const client = useQueryClient();
    const site = sitePath.exec(location.pathname)?.[1] ?? readLastSite();
    const listed = usePageTypes(site);
    const found = listed.data?.types?.find((type) => type.id === id) ?? cachedType(client, id);
    return <Resolved label={found?.name ?? null} pending={found === null && listed.isPending && site !== null} id={id} />;
}

function ScheduleRef({ id }: { id: string }): ReactElement {
    const detail = useSchedule(id);
    return <Resolved label={detail.data?.schedule.name ?? null} pending={detail.isPending} id={id} />;
}

function RunRef({ id }: { id: string }): ReactElement {
    const detail = useRun(id);
    const run = detail.data?.run;
    return <Resolved label={run === undefined ? null : `${run.kind} · ${run.status}`} pending={detail.isPending} id={id} mono={true} />;
}

export function RefLabel({ kind, id }: Ref): ReactElement {
    switch (kind) {
        case "page":
            return <PageRef id={id} />;
        case "entity":
            return <EntityRef id={id} />;
        case "template":
            return <TemplateRef id={id} />;
        case "site":
            return <SiteRef id={id} />;
        case "pagetype":
            return <PageTypeRef id={id} />;
        case "schedule":
            return <ScheduleRef id={id} />;
        case "run":
            return <RunRef id={id} />;
        default:
            return (
                <span className="font-mono text-ink-soft" title={id}>
                    {id.slice(0, shortIdLength)}
                </span>
            );
    }
}
