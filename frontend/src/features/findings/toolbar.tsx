import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../copy/index.js";
import { usePage } from "../../data/hooks/pages.js";
import { usePageTypes } from "../../data/hooks/templates.js";
import { findingStatuses } from "../../generated/vocab.js";
import type { FindingStatus } from "../../generated/vocab.js";
import type { SelectOption } from "../../ui/index.js";
import { AccountTreeIcon, Button, CloseIcon, Dialog, Segmented, Select, TaskAltIcon, Toolbar } from "../../ui/index.js";
import { optionLabel, statusLabel } from "./labels.js";
import type { FindingsQuery } from "./model/params.js";
import { PageSearch } from "./page-search.js";

const said = copy.findings;

const everyType = "every";

function UnderChip({ pageId, onClear }: { pageId: string; onClear: () => void }): ReactElement {
    const page = usePage(pageId);
    const path = page.data?.page.path ?? pageId;
    return (
        <span className="flex h-6 min-w-0 items-center gap-1 rounded-sm bg-accent-soft pr-0.5 pl-2 text-xs text-ink" title={said.underTooltip}>
            <AccountTreeIcon size={13} className="shrink-0 text-ink-faint" />
            <span className="truncate font-mono">{path}</span>
            <button
                type="button"
                aria-label={said.underRemove(path)}
                title={said.underRemove(path)}
                onClick={onClear}
                className="flex size-5 shrink-0 items-center justify-center rounded-sm text-ink-dim hover:bg-inset hover:text-ink"
            >
                <CloseIcon size={13} />
            </button>
        </span>
    );
}

function UnderPick({ siteId, onPick }: { siteId: string; onPick: (pageId: string) => void }): ReactElement {
    const [open, setOpen] = useState(false);
    const [picked, setPicked] = useState<string | null>(null);
    return (
        <>
            <Button
                size="sm"
                variant="ghost"
                icon={AccountTreeIcon}
                title={said.underTooltip}
                onClick={() => {
                    setPicked(null);
                    setOpen(true);
                }}
            >
                {said.under}
            </Button>
            <Dialog
                open={open}
                onOpenChange={setOpen}
                title={said.pick.title}
                description={said.pick.body}
                confirmLabel={said.pick.confirm}
                cancelLabel={said.pick.cancel}
                icon={AccountTreeIcon}
                refusal={picked === null ? copy.findings.resolve.pageFirst : null}
                onConfirm={() => {
                    if (picked !== null) {
                        onPick(picked);
                        setOpen(false);
                    }
                }}
            >
                <PageSearch siteId={siteId} picked={picked} onPick={(page) => setPicked(page.pageId)} />
            </Dialog>
        </>
    );
}

export interface BulkOffer {
    options: readonly string[];
    count: number;
    onBulk: (option: string) => void;
}

export interface FindingsToolbarProps {
    siteId: string;
    query: FindingsQuery;
    bulk: BulkOffer;
    onChange: (next: FindingsQuery) => void;
}

export function FindingsToolbar({ siteId, query, bulk, onChange }: FindingsToolbarProps): ReactElement {
    const types = usePageTypes(siteId);
    const typeOptions: SelectOption<string>[] = [
        { value: everyType, label: said.everyType },
        ...(types.data?.types ?? []).map((type) => ({ value: type.id, label: type.name })),
    ];
    return (
        <Toolbar label={said.title}>
            <Segmented<FindingStatus>
                label={said.status}
                value={query.status}
                options={findingStatuses.map((status) => ({ value: status, label: statusLabel(status) }))}
                onValueChange={(status) => {
                    onChange({ ...query, status });
                }}
            />
            <div className="w-44">
                <Select
                    size="sm"
                    value={query.type === "" ? everyType : query.type}
                    options={typeOptions}
                    aria-label={said.type}
                    onValueChange={(next) => {
                        onChange({ ...query, type: next === everyType ? "" : next });
                    }}
                />
            </div>
            {query.under === "" ? (
                <UnderPick
                    siteId={siteId}
                    onPick={(under) => {
                        onChange({ ...query, under });
                    }}
                />
            ) : (
                <UnderChip
                    pageId={query.under}
                    onClear={() => {
                        onChange({ ...query, under: "" });
                    }}
                />
            )}
            <div className="ml-auto flex shrink-0 items-center gap-1.5">
                {bulk.options.map((option) => (
                    <Button
                        key={option}
                        size="sm"
                        variant="secondary"
                        icon={TaskAltIcon}
                        title={said.bulk.tooltip}
                        data-finding-bulk={option}
                        onClick={() => {
                            bulk.onBulk(option);
                        }}
                    >
                        {said.bulk.all(optionLabel(option), bulk.count)}
                    </Button>
                ))}
            </div>
        </Toolbar>
    );
}
