import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import { react } from "../../data/errors.js";
import type { ImportField } from "../../generated/vocab.js";
import {
    Banner,
    Button,
    DenseTable,
    EmptyState,
    Select,
    SkeletonRows,
    TableCell,
    TableHead,
    TableRow,
    TableChartIcon,
    Tabs,
    VerifiedIcon,
} from "../../ui/index.js";
import type { ColumnMap } from "./columns.js";
import { takenFrom, targetOf, unmappedHeaders } from "./columns.js";
import { fieldChoices, noField } from "./labels.js";
import type { ColumnsNotice } from "./workbook.js";

const grid = "minmax(0,1.1fr) minmax(0,2fr) 10rem";

interface SheetTab {
    name: string;
    rows: number;
}

export interface StepColumnsProps {
    tabs: readonly SheetTab[];
    active: string;
    headers: readonly string[];
    sample: readonly (readonly string[] | null)[];
    busy: boolean;
    failure: unknown;
    columns: ColumnMap | null;
    detected: ColumnMap | null;
    notice: ColumnsNotice;
    onActivate: (sheet: string) => void;
    onAssign: (header: string, field: ImportField | null) => void;
    onBack: () => void;
    onNext: () => void;
}

function samplesOf(sample: readonly (readonly string[] | null)[], at: number): string {
    const cells: string[] = [];
    for (const row of sample) {
        const cell = row?.[at] ?? "";
        if (cell !== "") {
            cells.push(cell);
        }
        if (cells.length === 3) {
            break;
        }
    }
    return cells.join(" · ");
}

function blockedBy(notice: ColumnsNotice): string | undefined {
    switch (notice.kind) {
        case "ready":
            return undefined;
        case "noSheet":
            return copy.imports.columns.noSheet;
        case "otherSheet":
            return copy.imports.columns.otherSheet(notice.sheet);
        default:
            return copy.imports.columns.needsTarget;
    }
}

interface NoticeProps {
    notice: ColumnsNotice;
    ignored: number;
    onActivate: (sheet: string) => void;
}

function Notice({ notice, ignored, onActivate }: NoticeProps): ReactElement | null {
    switch (notice.kind) {
        case "ready":
            return ignored === 0 ? null : (
                <span className="min-w-0 flex-1 truncate text-xs text-ink-dim">
                    {copy.imports.columns.unmapped(ignored)}
                </span>
            );
        case "noSheet":
            return null;
        case "unmatched":
            return (
                <Banner
                    tone="warn"
                    title={copy.imports.columns.noHeaderHelp}
                    body={copy.imports.columns.noHeaderHelpBody}
                    className="min-w-0 flex-1"
                />
            );
        case "needsTarget":
            return <Banner tone="warn" title={copy.imports.columns.needsTarget} className="min-w-0 flex-1" />;
        case "otherSheet":
            return (
                <Banner
                    tone="warn"
                    title={copy.imports.columns.otherSheet(notice.sheet)}
                    className="min-w-0 flex-1"
                    actions={
                        <Button
                            size="sm"
                            variant="secondary"
                            onClick={() => {
                                onActivate(notice.sheet);
                            }}
                        >
                            {copy.imports.columns.openSheet(notice.sheet)}
                        </Button>
                    }
                />
            );
    }
}

interface ColumnTableProps {
    headers: readonly string[];
    sample: readonly (readonly string[] | null)[];
    busy: boolean;
    failure: unknown;
    columns: ColumnMap | null;
    detected: ColumnMap | null;
    chosen: boolean;
    onAssign: (header: string, field: ImportField | null) => void;
}

function ColumnTable({ headers, sample, busy, failure, columns, detected, chosen, onAssign }: ColumnTableProps): ReactElement {
    if (!chosen) {
        return (
            <div className="p-4">
                <EmptyState icon={TableChartIcon} title={copy.imports.columns.noSheet} />
            </div>
        );
    }
    if (headers.length === 0) {
        const reaction = failure === null || failure === undefined ? null : react(failure);
        if (busy) {
            return (
                <div className="p-4">
                    <SkeletonRows rows={8} label={copy.imports.file.reading} />
                </div>
            );
        }
        if (reaction !== null && reaction.kind !== "silent" && reaction.kind !== "unlock") {
            return (
                <div className="p-4">
                    <Banner tone="danger" title={reaction.message} />
                </div>
            );
        }
        return (
            <div className="p-4">
                <EmptyState icon={TableChartIcon} title={copy.imports.columns.empty} />
            </div>
        );
    }
    return (
        <DenseTable columns={grid} label={copy.imports.columns.title}>
            <TableHead>
                <span>{copy.imports.columns.header}</span>
                <span>{copy.imports.columns.sample}</span>
                <span>{copy.imports.columns.target}</span>
            </TableHead>
            {headers.map((header, at) => {
                const target = targetOf(columns, header);
                return (
                    <TableRow key={`${header}-${at}`}>
                        <TableCell mono={true} title={header}>
                            <span className="flex min-w-0 items-center gap-1.5">
                                <span className="truncate">{header}</span>
                                {takenFrom(columns, detected, header) ? (
                                    <VerifiedIcon
                                        size={13}
                                        className="shrink-0 text-ok"
                                        aria-label={copy.imports.columns.detected}
                                    />
                                ) : null}
                            </span>
                        </TableCell>
                        <TableCell mono={true} muted={true} title={samplesOf(sample, at)}>
                            {samplesOf(sample, at)}
                        </TableCell>
                        <TableCell>
                            <Select
                                aria-label={copy.imports.columns.target}
                                size="sm"
                                quiet={target === null}
                                value={target ?? noField}
                                options={fieldChoices}
                                onValueChange={(next) => {
                                    onAssign(header, next === noField ? null : next);
                                }}
                            />
                        </TableCell>
                    </TableRow>
                );
            })}
        </DenseTable>
    );
}

export function StepColumns({
    tabs,
    active,
    headers,
    sample,
    busy,
    failure,
    columns,
    detected,
    notice,
    onActivate,
    onAssign,
    onBack,
    onNext,
}: StepColumnsProps): ReactElement {
    const blocked = blockedBy(notice);

    return (
        <div className="flex min-h-0 flex-1 flex-col">
            {tabs.length === 0 ? null : (
                <div className="flex h-8 shrink-0 items-stretch border-b border-hairline px-2">
                    <Tabs<string>
                        label={copy.imports.columns.sheetTabs}
                        value={active}
                        items={tabs.map((tab) => ({ key: tab.name, label: tab.name, count: tab.rows }))}
                        onValueChange={onActivate}
                    />
                </div>
            )}
            <div className="min-h-0 flex-1 overflow-auto">
                <ColumnTable
                    headers={headers}
                    sample={sample}
                    busy={busy}
                    failure={failure}
                    columns={columns}
                    detected={detected}
                    chosen={notice.kind !== "noSheet"}
                    onAssign={onAssign}
                />
            </div>
            <div className="flex shrink-0 items-center gap-3 border-t border-hairline p-3">
                <Notice notice={notice} ignored={unmappedHeaders(headers, columns).length} onActivate={onActivate} />
                <div className="ml-auto flex shrink-0 gap-2">
                    <Button variant="secondary" onClick={onBack}>
                        {copy.imports.back}
                    </Button>
                    <Button
                        variant="primary"
                        disabled={blocked !== undefined}
                        title={blocked}
                        onClick={onNext}
                    >
                        {copy.imports.next}
                    </Button>
                </div>
            </div>
        </div>
    );
}
