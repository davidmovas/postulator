import type { ReactElement } from "react";
import { useMemo, useState } from "react";

import { copy } from "../../copy/index.js";
import type { PreviewReport } from "../../data/types.js";
import { importActions } from "../../generated/vocab.js";
import {
    Banner,
    Button,
    DenseTable,
    Segmented,
    StatusBadge,
    TableCell,
    TableHead,
    TableRow,
    cx,
    toneClasses,
} from "../../ui/index.js";
import { actionLabel, actionTone, edgeKindLabel, entityKindLabel } from "./labels.js";

type Sheet = "pages" | "entities" | "edges";

const grids: Readonly<Record<Sheet, string>> = {
    pages: "minmax(0,2fr) minmax(0,1.6fr) minmax(0,1fr) 6rem",
    entities: "minmax(0,1.4fr) 7rem minmax(0,1.4fr) 6rem",
    edges: "minmax(0,1.4fr) minmax(0,1.4fr) 7rem 6rem",
};

interface Countable {
    action: string;
}

function tally(rows: readonly Countable[]): { action: string; count: number }[] {
    return importActions
        .map((action) => ({ action, count: rows.filter((row) => row.action === action).length }))
        .filter((held) => held.count > 0);
}

export interface StepPreviewProps {
    report: PreviewReport;
    onBack: () => void;
    onApply: () => void;
}

export function StepPreview({ report, onBack, onApply }: StepPreviewProps): ReactElement {
    const [sheet, setSheet] = useState<Sheet>("pages");
    const pages = useMemo(() => report.pages ?? [], [report.pages]);
    const entities = useMemo(() => report.entities ?? [], [report.entities]);
    const edges = useMemo(() => report.edges ?? [], [report.edges]);
    const errors = report.errors ?? [];
    const counted = tally(sheet === "pages" ? pages : sheet === "entities" ? entities : edges);

    return (
        <div className="flex min-h-0 flex-1 flex-col">
            <div className="flex shrink-0 flex-wrap items-center gap-2 border-b border-hairline px-3 py-2">
                <Segmented
                    label={copy.imports.preview.title}
                    value={sheet}
                    options={[
                        { value: "pages", label: `${copy.imports.preview.pages} ${pages.length}` },
                        { value: "entities", label: `${copy.imports.preview.entities} ${entities.length}` },
                        { value: "edges", label: `${copy.imports.preview.edges} ${edges.length}` },
                    ]}
                    onValueChange={setSheet}
                />
                <div className="flex flex-wrap items-center gap-1.5">
                    {counted.map((held) => (
                        <span
                            key={held.action}
                            className={cx(
                                "flex h-5 items-center gap-1 rounded-sm px-1.5 text-2xs",
                                toneClasses[actionTone(held.action)].soft,
                                toneClasses[actionTone(held.action)].ink,
                            )}
                        >
                            <span className="font-mono">{held.count}</span>
                            {actionLabel(held.action)}
                        </span>
                    ))}
                    {report.skipped === 0 ? null : (
                        <span className="text-2xs text-ink-faint">{copy.imports.preview.skipped(report.skipped)}</span>
                    )}
                </div>
            </div>
            <div className="min-h-0 flex-1 overflow-auto">
                <DenseTable columns={grids[sheet]} label={copy.imports.preview.title}>
                    <TableHead>
                        {sheet === "pages" ? (
                            <>
                                <span>{copy.imports.preview.columnPath}</span>
                                <span>{copy.imports.preview.columnTitle}</span>
                                <span>{copy.imports.preview.columnEntity}</span>
                                <span>{copy.imports.preview.columnAction}</span>
                            </>
                        ) : sheet === "entities" ? (
                            <>
                                <span>{copy.imports.preview.columnName}</span>
                                <span>{copy.imports.preview.columnKind}</span>
                                <span>{copy.imports.preview.columnKeyword}</span>
                                <span>{copy.imports.preview.columnAction}</span>
                            </>
                        ) : (
                            <>
                                <span>{copy.imports.preview.columnFrom}</span>
                                <span>{copy.imports.preview.columnTo}</span>
                                <span>{copy.imports.preview.columnRelation}</span>
                                <span>{copy.imports.preview.columnAction}</span>
                            </>
                        )}
                    </TableHead>
                    {sheet === "pages"
                        ? pages.map((page, at) => (
                              <TableRow key={`${page.path}-${at}`}>
                                  <TableCell mono={true} title={page.path}>
                                      {page.path}
                                  </TableCell>
                                  <TableCell title={page.title}>{page.title}</TableCell>
                                  <TableCell muted={true}>{page.entity ?? ""}</TableCell>
                                  <TableCell>
                                      <StatusBadge tone={actionTone(page.action)} dot={false}>
                                          {actionLabel(page.action)}
                                      </StatusBadge>
                                  </TableCell>
                              </TableRow>
                          ))
                        : sheet === "entities"
                          ? entities.map((entity, at) => (
                                <TableRow key={`${entity.name}-${at}`}>
                                    <TableCell title={entity.name}>{entity.name}</TableCell>
                                    <TableCell muted={true}>{entityKindLabel(entity.kind)}</TableCell>
                                    <TableCell mono={true} muted={true}>
                                        {entity.primaryKeyword ?? ""}
                                    </TableCell>
                                    <TableCell>
                                        <StatusBadge tone={actionTone(entity.action)} dot={false}>
                                            {actionLabel(entity.action)}
                                        </StatusBadge>
                                    </TableCell>
                                </TableRow>
                            ))
                          : edges.map((edge, at) => (
                                <TableRow key={`${edge.from}-${edge.to}-${at}`}>
                                    <TableCell title={edge.from}>{edge.from}</TableCell>
                                    <TableCell title={edge.to}>{edge.to}</TableCell>
                                    <TableCell muted={true}>{edgeKindLabel(edge.kind)}</TableCell>
                                    <TableCell>
                                        <StatusBadge tone={actionTone(edge.action)} dot={false}>
                                            {actionLabel(edge.action)}
                                        </StatusBadge>
                                    </TableCell>
                                </TableRow>
                            ))}
                </DenseTable>
            </div>
            <div className="flex shrink-0 items-center gap-3 border-t border-hairline p-3">
                {errors.length === 0 ? null : (
                    <Banner tone="danger" title={copy.imports.preview.blocked} className="min-w-0 flex-1" />
                )}
                <div className="ml-auto flex shrink-0 gap-2">
                    <Button variant="secondary" onClick={onBack}>
                        {copy.imports.back}
                    </Button>
                    <Button
                        variant="primary"
                        data-import-apply={true}
                        disabled={errors.length > 0}
                        title={errors.length > 0 ? copy.imports.preview.blocked : undefined}
                        onClick={onApply}
                    >
                        {copy.imports.apply.start}
                    </Button>
                </div>
            </div>
        </div>
    );
}
