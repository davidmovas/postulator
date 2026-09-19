import type { ReactElement } from "react";
import { useEffect, useState } from "react";
import { Link } from "react-router";

import { useEntity } from "../../data/hooks/graph.js";
import { useMapPageToEntity, useSetCanonicalPage, useUnmapPage } from "../../data/hooks/pages.js";
import type { Page } from "../../data/types.js";
import { copy } from "../../copy/index.js";
import type { SelectOption } from "../../ui/index.js";
import {
    Button,
    LinkOffIcon,
    Panel,
    PanelHeader,
    Select,
    Skeleton,
    StatusBadge,
    VerifiedIcon,
} from "../../ui/index.js";
import { ConflictNotice } from "./conflict-notice.js";
import type { EntityIndex } from "./entities.js";
import { entityIcon } from "./labels.js";
import { usePageDetailRefresh } from "./refresh.js";

const noEntity = "none";

export interface PageMappingProps {
    page: Page;
    siteId: string;
    index: EntityIndex;
    search: string;
}

export function PageMapping({ page, siteId, index, search }: PageMappingProps): ReactElement {
    const entityQuery = useEntity(page.entityId);
    const map = useMapPageToEntity();
    const unmap = useUnmapPage();
    const canonical = useSetCanonicalPage();
    const refresh = usePageDetailRefresh();
    const [choice, setChoice] = useState(page.entityId ?? noEntity);
    const settle = {
        onSuccess: () => {
            refresh(page.id);
        },
    };

    useEffect(() => {
        setChoice(page.entityId ?? noEntity);
        map.reset();
    }, [page.id, page.entityId]);

    const entity = entityQuery.data?.entity;
    const Icon = entityIcon(entity?.kind ?? "");
    const isCanonical = entity !== undefined && entity.canonicalPageId === page.id;
    const canonicalElsewhere =
        entity !== undefined && entity.canonicalPageId !== null && entity.canonicalPageId !== page.id;

    const options: SelectOption<string>[] = [
        { value: noEntity, label: copy.pages.detail.noEntityOption },
        ...index.entities.map((held) => ({ value: held.id, label: held.name })),
    ];

    const changed = choice !== (page.entityId ?? noEntity);

    return (
        <Panel>
            <PanelHeader title={copy.pages.detail.entity}>
                {isCanonical ? (
                    <StatusBadge tone="ok" icon={VerifiedIcon}>
                        {copy.pages.detail.canonical}
                    </StatusBadge>
                ) : null}
            </PanelHeader>
            <div className="flex flex-col gap-2.5 p-3">
                {page.entityId === null ? (
                    <div className="flex items-start gap-2">
                        <LinkOffIcon size={18} className="mt-px shrink-0 text-ink-faint" />
                        <div className="flex min-w-0 flex-col gap-0.5">
                            <p className="text-sm font-semibold text-ink">{copy.pages.detail.noEntity}</p>
                            <p className="text-xs text-ink-dim">{copy.pages.detail.noEntityBody}</p>
                        </div>
                    </div>
                ) : entityQuery.isPending ? (
                    <Skeleton height={18} width="60%" />
                ) : entity === undefined ? (
                    <p className="text-xs text-ink-dim">{copy.pages.detail.entityMissing}</p>
                ) : (
                    <div className="flex items-center gap-2">
                        <Icon size={18} className="shrink-0 text-accent" />
                        <div className="flex min-w-0 flex-col">
                            <span className="truncate text-sm font-semibold text-ink">{entity.name}</span>
                            <span className="font-mono text-2xs text-ink-faint">
                                {copy.pages.detail.entityMeta(entity.kind, entity.score)}
                            </span>
                        </div>
                    </div>
                )}

                <div className="flex items-end gap-2">
                    <Select
                        className="min-w-0 flex-1"
                        value={choice}
                        options={options}
                        aria-label={copy.pages.detail.entity}
                        disabled={index.entities.length === 0}
                        onValueChange={setChoice}
                    />
                    {choice === noEntity ? (
                        <Button
                            variant="secondary"
                            disabled={page.entityId === null}
                            busy={unmap.isPending}
                            onClick={() => {
                                unmap.mutate({ pageId: page.id }, settle);
                            }}
                        >
                            {copy.pages.detail.unmap}
                        </Button>
                    ) : (
                        <Button
                            variant="primary"
                            disabled={!changed}
                            busy={map.isPending}
                            onClick={() => {
                                map.mutate({ pageId: page.id, entityId: choice }, settle);
                            }}
                        >
                            {page.entityId === null ? copy.pages.detail.map : copy.pages.detail.change}
                        </Button>
                    )}
                </div>
                {index.entities.length === 0 ? (
                    <p className="text-2xs text-ink-faint">{copy.pages.filters.noEntities}</p>
                ) : null}

                <ConflictNotice thrown={map.error} siteId={siteId} search={search} />

                <div className="flex flex-col gap-1 rounded-md border border-hairline bg-inset p-2.5">
                    <p className="text-xs font-semibold text-ink">{copy.pages.detail.canonicalTitle}</p>
                    <p className="text-2xs text-ink-dim">{copy.pages.detail.canonicalBody}</p>
                    {entity === undefined ? null : isCanonical ? (
                        <p className="text-xs text-ok">{copy.pages.detail.canonicalIsHere}</p>
                    ) : (
                        <div className="flex flex-wrap items-center gap-2">
                            <p className="text-xs text-warn">
                                {canonicalElsewhere
                                    ? copy.pages.detail.canonicalElsewhere
                                    : copy.pages.detail.canonicalMissing}
                            </p>
                            {canonicalElsewhere && entity.canonicalPageId !== null ? (
                                <Link
                                    to={`/s/${siteId}/pages/${entity.canonicalPageId}${search}`}
                                    className="text-xs"
                                >
                                    {copy.pages.detail.openCanonical}
                                </Link>
                            ) : null}
                            <Button
                                size="sm"
                                variant="primary"
                                busy={canonical.isPending}
                                onClick={() => {
                                    canonical.mutate({ entityId: entity.id, pageId: page.id }, settle);
                                }}
                            >
                                {copy.pages.detail.makeCanonical}
                            </Button>
                        </div>
                    )}
                </div>
            </div>
        </Panel>
    );
}
