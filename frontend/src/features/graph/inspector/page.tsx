import type { ReactElement } from "react";
import { useMemo, useState } from "react";
import { Link } from "react-router";

import { copy } from "../../../copy/index.js";
import { flatten } from "../../../data/call.js";
import { useMapPageToEntity, usePage, usePages, useSetCanonicalPage } from "../../../data/hooks/pages.js";
import type { Entity } from "../../../data/types.js";
import { Banner, Button, Input, LinkOffIcon, Select, Skeleton, StatusBadge } from "../../../ui/index.js";
import type { SelectOption } from "../../../ui/index.js";
import { pageStatusLabel, statusTone } from "../../pages/labels.js";
import { formErrorOf } from "./fields.js";

const pickLimit = 100;
const findLimit = 50;

export interface CanonicalPageProps {
    siteId: string;
    entity: Entity;
    onPlan: () => void;
}

function CurrentPage({ siteId, pageId }: { siteId: string; pageId: string }): ReactElement {
    const page = usePage(pageId);
    const held = page.data?.page;
    if (page.isPending) {
        return <Skeleton height={18} width="70%" />;
    }
    return (
        <div className="flex flex-col gap-1">
            <Link to={`/s/${siteId}/pages/${pageId}`} className="truncate font-mono text-xs" title={held?.path ?? ""}>
                {held?.path ?? copy.graph.pageForm.open}
            </Link>
            {held === undefined ? null : (
                <span className="flex items-center gap-2">
                    <StatusBadge tone={statusTone(held.status)}>{pageStatusLabel(held.status)}</StatusBadge>
                    {held.drift ? (
                        <StatusBadge tone="warn" dot={false}>
                            {copy.graph.pageForm.drift}
                        </StatusBadge>
                    ) : null}
                </span>
            )}
        </div>
    );
}

function ChoosePage({ siteId, entity, onPlan }: CanonicalPageProps): ReactElement {
    const mapped = usePages({ siteId, entityId: entity.id }, null, pickLimit);
    const [prefix, setPrefix] = useState("");
    const unmapped = usePages({ siteId, unmapped: true, pathPrefix: prefix }, { field: "path", desc: false }, findLimit);
    const makeCanonical = useSetCanonicalPage();
    const map = useMapPageToEntity();
    const [choice, setChoice] = useState("");
    const [candidate, setCandidate] = useState("");

    const mappedPages = useMemo(() => flatten(mapped.data?.pages), [mapped.data]);
    const unmappedPages = useMemo(() => flatten(unmapped.data?.pages), [unmapped.data]);
    const mappedOptions: SelectOption<string>[] = mappedPages.map((page) => ({ value: page.id, label: page.path }));
    const candidateOptions: SelectOption<string>[] = unmappedPages.map((page) => ({ value: page.id, label: page.path }));
    const chosen = choice !== "" && mappedPages.some((page) => page.id === choice) ? choice : (mappedPages[0]?.id ?? "");
    const picked = candidate !== "" && unmappedPages.some((page) => page.id === candidate) ? candidate : (unmappedPages[0]?.id ?? "");

    const mapAndMakeCanonical = async (): Promise<void> => {
        if (picked === "") {
            return;
        }
        await map.mutateAsync({ pageId: picked, entityId: entity.id });
        await makeCanonical.mutateAsync({ entityId: entity.id, pageId: picked });
    };

    const error = formErrorOf(makeCanonical.error) ?? formErrorOf(map.error);

    return (
        <div className="flex flex-col gap-3">
            <Banner tone="danger" icon={LinkOffIcon} title={copy.graph.inspector.noPage} body={copy.graph.inspector.noPageBody} />
            <div className="flex flex-col gap-1.5">
                <p className="text-2xs text-ink-faint">{copy.graph.pageForm.mapped}</p>
                {mapped.isPending ? (
                    <Skeleton height={28} />
                ) : mappedPages.length === 0 ? (
                    <p className="text-xs text-ink-dim">{copy.graph.pageForm.none}</p>
                ) : (
                    <div className="flex items-center gap-2">
                        <div className="min-w-0 flex-1">
                            <Select value={chosen} options={mappedOptions} aria-label={copy.graph.pageForm.mapped} onValueChange={setChoice} />
                        </div>
                        <Button
                            size="sm"
                            variant="primary"
                            busy={makeCanonical.isPending}
                            onClick={() => {
                                makeCanonical.mutate({ entityId: entity.id, pageId: chosen });
                            }}
                        >
                            {copy.graph.pageForm.makeCanonical}
                        </Button>
                    </div>
                )}
            </div>
            <div className="flex flex-col gap-1.5">
                <Input
                    mono={true}
                    aria-label={copy.graph.pageForm.find}
                    placeholder={copy.graph.pageForm.find}
                    value={prefix}
                    onChange={(event) => {
                        setPrefix(event.target.value);
                    }}
                />
                {unmapped.isPending ? (
                    <Skeleton height={28} />
                ) : unmappedPages.length === 0 ? (
                    <p className="text-xs text-ink-dim">{copy.graph.pageForm.noMatch}</p>
                ) : (
                    <div className="flex items-center gap-2">
                        <div className="min-w-0 flex-1">
                            <Select value={picked} options={candidateOptions} aria-label={copy.graph.pageForm.find} onValueChange={setCandidate} />
                        </div>
                        <Button
                            size="sm"
                            variant="secondary"
                            busy={map.isPending || makeCanonical.isPending}
                            onClick={() => {
                                void mapAndMakeCanonical();
                            }}
                        >
                            {copy.graph.pageForm.map}
                        </Button>
                    </div>
                )}
                {unmapped.hasNextPage ? <p className="text-2xs text-ink-faint">{copy.graph.pageForm.partial}</p> : null}
            </div>
            {error === null ? null : <p className="text-xs text-danger">{error}</p>}
            <Button size="sm" variant="ghost" onClick={onPlan}>
                {copy.graph.pageForm.plan}
            </Button>
        </div>
    );
}

export function CanonicalPage({ siteId, entity, onPlan }: CanonicalPageProps): ReactElement {
    if (entity.canonicalPageId !== null) {
        return <CurrentPage siteId={siteId} pageId={entity.canonicalPageId} />;
    }
    return <ChoosePage siteId={siteId} entity={entity} onPlan={onPlan} />;
}
