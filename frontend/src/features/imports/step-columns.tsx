import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { ImportField } from "../../generated/vocab.js";
import {
    Banner,
    Button,
    DenseTable,
    EmptyState,
    Select,
    TableCell,
    TableHead,
    TableRow,
    TableChartIcon,
    VerifiedIcon,
} from "../../ui/index.js";
import type { ColumnMap } from "./columns.js";
import { takenFrom, targetOf, unmappedHeaders, usable } from "./columns.js";
import { fieldChoices, noField } from "./labels.js";

const grid = "minmax(0,1.1fr) minmax(0,2fr) 13rem";

export interface StepColumnsProps {
    headers: readonly string[];
    sample: readonly (readonly string[] | null)[];
    columns: ColumnMap | null;
    detected: ColumnMap | null;
    indentColumns: readonly string[];
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

export function StepColumns({
    headers,
    sample,
    columns,
    detected,
    indentColumns,
    onAssign,
    onBack,
    onNext,
}: StepColumnsProps): ReactElement {
    const ignored = unmappedHeaders(headers, columns);
    const ready = usable(columns, indentColumns);

    if (headers.length === 0) {
        return (
            <div className="p-4">
                <EmptyState icon={TableChartIcon} title={copy.imports.columns.empty} />
            </div>
        );
    }

    return (
        <div className="flex min-h-0 flex-1 flex-col">
            <div className="min-h-0 flex-1 overflow-auto">
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
            </div>
            <div className="flex shrink-0 items-center gap-3 border-t border-hairline p-3">
                {ready ? (
                    ignored.length === 0 ? null : (
                        <span className="min-w-0 flex-1 truncate text-xs text-ink-dim">
                            {copy.imports.columns.unmapped(ignored.length)}
                        </span>
                    )
                ) : (
                    <Banner tone="warn" title={copy.imports.columns.needsTarget} className="min-w-0 flex-1" />
                )}
                <div className="ml-auto flex shrink-0 gap-2">
                    <Button variant="secondary" onClick={onBack}>
                        {copy.imports.back}
                    </Button>
                    <Button
                        variant="primary"
                        disabled={!ready}
                        title={ready ? undefined : copy.imports.columns.needsTarget}
                        onClick={onNext}
                    >
                        {copy.imports.next}
                    </Button>
                </div>
            </div>
        </div>
    );
}
