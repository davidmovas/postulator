import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../copy/index.js";
import { react } from "../../data/errors.js";
import { useDeleteMapping, useMappings } from "../../data/hooks/imports.js";
import type { ImportMapping } from "../../data/types.js";
import { relativeTime } from "../../domain/format.js";
import {
    Banner,
    Button,
    DeleteIcon,
    Dialog,
    EmptyState,
    Menu,
    MoreHorizIcon,
    Panel,
    PanelHeader,
    SkeletonRows,
    TableChartIcon,
} from "../../ui/index.js";
import { mappedFields } from "./columns.js";
import { fieldLabel } from "./labels.js";

export interface MappingsPanelProps {
    siteId: string;
    activeId: string;
    onUse: (mapping: ImportMapping) => void;
}

export function MappingsPanel({ siteId, activeId, onUse }: MappingsPanelProps): ReactElement {
    const mappings = useMappings(siteId === "" ? null : siteId);
    const remove = useDeleteMapping();
    const [doomed, setDoomed] = useState<ImportMapping | null>(null);
    const rows = mappings.data?.mappings ?? [];
    const failure = mappings.error === null ? null : react(mappings.error);

    return (
        <div className="flex flex-col gap-3 p-3">
            <Panel>
                <PanelHeader title={copy.imports.mappings.title} />
                <div className="flex flex-col gap-2 p-3">
                    {mappings.isPending ? (
                        <SkeletonRows rows={3} label={copy.app.loading} />
                    ) : failure !== null && failure.kind !== "silent" && failure.kind !== "unlock" ? (
                        <Banner tone="danger" title={failure.message} />
                    ) : rows.length === 0 ? (
                        <EmptyState icon={TableChartIcon} title={copy.empty.mappings} />
                    ) : (
                        rows.map((mapping) => (
                            <article
                                key={mapping.id}
                                className="flex flex-col gap-1.5 rounded-md border border-hairline bg-inset p-2.5"
                            >
                                <div className="flex items-center gap-1.5">
                                    <h3 className="min-w-0 flex-1 truncate text-xs font-semibold text-ink">
                                        {mapping.name}
                                    </h3>
                                    <Menu
                                        label={copy.app.more}
                                        align="end"
                                        trigger={
                                            <button
                                                type="button"
                                                aria-label={copy.app.more}
                                                className="flex h-5 w-5 shrink-0 items-center justify-center rounded-sm text-ink-dim hover:bg-raised hover:text-ink"
                                            >
                                                <MoreHorizIcon size={14} />
                                            </button>
                                        }
                                        items={[
                                            {
                                                key: "delete",
                                                label: copy.imports.mappings.delete,
                                                icon: DeleteIcon,
                                                danger: true,
                                                onSelect: () => {
                                                    setDoomed(mapping);
                                                },
                                            },
                                        ]}
                                    />
                                </div>
                                <p className="truncate font-mono text-2xs text-ink-faint">
                                    {mappedFields(mapping.columns).map(fieldLabel).join(", ")}
                                </p>
                                <div className="flex items-center gap-2">
                                    <span className="min-w-0 flex-1 truncate text-2xs text-ink-faint">
                                        {copy.imports.mappings.updated(relativeTime(mapping.updatedAt))}
                                    </span>
                                    <Button
                                        size="sm"
                                        variant={mapping.id === activeId ? "ghost" : "secondary"}
                                        disabled={mapping.id === activeId}
                                        onClick={() => {
                                            onUse(mapping);
                                        }}
                                    >
                                        {mapping.id === activeId
                                            ? copy.imports.mappings.inUse
                                            : copy.imports.mappings.use}
                                    </Button>
                                </div>
                            </article>
                        ))
                    )}
                </div>
            </Panel>
            <Dialog
                open={doomed !== null}
                onOpenChange={(open) => {
                    if (!open) {
                        setDoomed(null);
                    }
                }}
                title={copy.imports.mappings.delete}
                description={doomed?.name ?? ""}
                confirmLabel={copy.imports.mappings.delete}
                cancelLabel={copy.app.cancel}
                destructive={true}
                busy={remove.isPending}
                onConfirm={() => {
                    const target = doomed;
                    if (target?.id === undefined) {
                        return;
                    }
                    remove.mutate(
                        { id: target.id },
                        {
                            onSuccess: () => {
                                setDoomed(null);
                            },
                        },
                    );
                }}
            />
        </div>
    );
}
