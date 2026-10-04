import type { ReactElement } from "react";
import { useMemo, useState } from "react";

import { copy } from "../../copy/index.js";
import type { PreviewReport } from "../../data/types.js";
import { trailOf } from "../../domain/entities.js";
import { keywordsLine } from "../../domain/keywords.js";
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
import { entityKindLabel } from "../graph/labels.js";
import { actionLabel, actionTone, columnUseLabel, edgeKindLabel, productNote } from "./labels.js";
import type { PreviewColumn } from "./preview.js";
import { columnsBySheet, sheetsIn } from "./preview.js";

type Segment = "pages" | "entities" | "groups" | "edges";

const grids: Readonly<Record<Segment, string>> = {
    pages: "minmax(0,2fr) minmax(0,1.6fr) minmax(0,1.2fr) minmax(0,1fr) 6rem",
    entities: "minmax(0,1.8fr) 7rem minmax(0,1.4fr) 6rem",
    groups: "minmax(0,2fr) minmax(0,1.6fr) 5rem",
    edges: "minmax(0,1.4fr) minmax(0,1.4fr) 7rem 6rem",
};

const sheetTrack = "minmax(0,0.8fr)";

interface Countable {
    action: string;
}

function tally(rows: readonly Countable[]): { action: string; count: number }[] {
    return importActions
        .map((action) => ({ action, count: rows.filter((row) => row.action === action).length }))
        .filter((held) => held.count > 0);
}

interface SheetCellProps {
    shown: boolean;
    sheet: string | undefined;
}

function SheetCell({ shown, sheet }: SheetCellProps): ReactElement | null {
    if (!shown) {
        return null;
    }
    return (
        <TableCell muted={true} title={sheet ?? ""}>
            {sheet ?? ""}
        </TableCell>
    );
}

function ColumnChips({ columns }: { columns: readonly PreviewColumn[] }): ReactElement {
    return (
        <>
            {columns.map((column, at) => (
                <span
                    key={`${column.header}-${String(at)}`}
                    className={cx(
                        "flex h-5 items-center gap-1 rounded-sm bg-inset px-1.5 text-2xs",
                        column.use === "ignored" ? "text-ink-faint" : "text-ink-soft",
                    )}
                >
                    <span className="font-mono">{column.header === "" ? "—" : column.header}</span>
                    <span aria-hidden={true}>→</span>
                    {columnUseLabel(column)}
                </span>
            ))}
        </>
    );
}

function ColumnUses({ columns }: { columns: readonly PreviewColumn[] }): ReactElement | null {
    if (columns.length === 0) {
        return null;
    }
    const bySheet = columnsBySheet(columns);
    return (
        <section
            aria-label={copy.imports.preview.columns}
            className="flex shrink-0 flex-col gap-1.5 border-b border-hairline px-3 py-2"
        >
            {bySheet.length < 2 ? (
                <div className="flex flex-wrap items-center gap-1.5">
                    <span className="text-2xs text-ink-faint">{copy.imports.preview.columns}</span>
                    <ColumnChips columns={columns} />
                </div>
            ) : (
                <>
                    <span className="text-2xs text-ink-faint">{copy.imports.preview.columns}</span>
                    {bySheet.map((held) => (
                        <div key={held.sheet} className="flex flex-wrap items-center gap-1.5">
                            <span
                                className="w-24 shrink-0 truncate text-2xs font-medium text-ink-dim"
                                title={held.sheet}
                            >
                                {held.sheet}
                            </span>
                            <ColumnChips columns={held.columns} />
                        </div>
                    ))}
                </>
            )}
        </section>
    );
}

export interface StepPreviewProps {
    report: PreviewReport;
    onBack: () => void;
    onNext: () => void;
}

export function StepPreview({ report, onBack, onNext }: StepPreviewProps): ReactElement {
    const [segment, setSegment] = useState<Segment>("pages");
    const pages = useMemo(() => report.pages ?? [], [report.pages]);
    const entities = useMemo(() => report.entities ?? [], [report.entities]);
    const groups = useMemo(() => report.groups ?? [], [report.groups]);
    const edges = useMemo(() => report.edges ?? [], [report.edges]);
    const shown = useMemo(() => sheetsIn(report).length > 1, [report]);
    const errors = report.errors ?? [];
    const counted = ((): { action: string; count: number }[] => {
        switch (segment) {
            case "pages":
                return tally(pages);
            case "entities":
                return tally(entities);
            case "edges":
                return tally(edges);
            default:
                return [];
        }
    })();

    return (
        <div className="flex min-h-0 flex-1 flex-col">
            <div className="flex shrink-0 flex-wrap items-center gap-2 border-b border-hairline px-3 py-2">
                <Segmented
                    label={copy.imports.preview.title}
                    value={segment}
                    options={[
                        { value: "pages", label: `${copy.imports.preview.pages} ${pages.length}` },
                        { value: "entities", label: `${copy.imports.preview.entities} ${entities.length}` },
                        { value: "groups", label: `${copy.imports.preview.groups} ${groups.length}` },
                        { value: "edges", label: `${copy.imports.preview.edges} ${edges.length}` },
                    ]}
                    onValueChange={setSegment}
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
            <ColumnUses columns={report.columns ?? []} />
            <div className="min-h-0 flex-1 overflow-auto">
                <DenseTable
                    columns={shown ? `${sheetTrack} ${grids[segment]}` : grids[segment]}
                    label={copy.imports.preview.title}
                >
                    <TableHead>
                        {shown ? <span>{copy.imports.preview.columnSheet}</span> : null}
                        {segment === "pages" ? (
                            <>
                                <span>{copy.imports.preview.columnPath}</span>
                                <span>{copy.imports.preview.columnTitle}</span>
                                <span>{copy.imports.preview.columnKeyword}</span>
                                <span>{copy.imports.preview.columnEntity}</span>
                                <span>{copy.imports.preview.columnAction}</span>
                            </>
                        ) : segment === "entities" ? (
                            <>
                                <span>{copy.imports.preview.columnName}</span>
                                <span>{copy.imports.preview.columnKind}</span>
                                <span>{copy.imports.preview.columnKeyword}</span>
                                <span>{copy.imports.preview.columnAction}</span>
                            </>
                        ) : segment === "groups" ? (
                            <>
                                <span>{copy.imports.preview.columnGroup}</span>
                                <span>{copy.imports.preview.columnPage}</span>
                                <span>{copy.imports.preview.columnRows}</span>
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
                    {segment === "pages"
                        ? pages.map((page, at) => (
                              <TableRow key={`${page.sheet ?? ""}-${page.path}-${at}`}>
                                  <SheetCell shown={shown} sheet={page.sheet} />
                                  <TableCell mono={true} title={page.path}>
                                      {page.path}
                                  </TableCell>
                                  <TableCell
                                      muted={productNote(page) !== null}
                                      title={productNote(page) ?? page.title}
                                  >
                                      {productNote(page) ?? page.title}
                                  </TableCell>
                                  <TableCell muted={true} title={keywordsLine(page.keywords)}>
                                      {keywordsLine(page.keywords)}
                                  </TableCell>
                                  <TableCell muted={true}>{page.entity ?? ""}</TableCell>
                                  <TableCell>
                                      <StatusBadge tone={actionTone(page.action)} dot={false}>
                                          {actionLabel(page.action)}
                                      </StatusBadge>
                                  </TableCell>
                              </TableRow>
                          ))
                        : segment === "entities"
                          ? entities.map((entity, at) => (
                                <TableRow key={`${entity.sheet ?? ""}-${entity.name}-${at}`}>
                                    <SheetCell shown={shown} sheet={entity.sheet} />
                                    <TableCell title={trailOf([entity.parent ?? "", entity.name])}>
                                        {trailOf([entity.parent ?? "", entity.name])}
                                    </TableCell>
                                    <TableCell muted={true}>{entityKindLabel(entity.kind)}</TableCell>
                                    <TableCell mono={true} muted={true} title={keywordsLine(entity.keywords)}>
                                        {keywordsLine(entity.keywords)}
                                    </TableCell>
                                    <TableCell>
                                        <StatusBadge tone={actionTone(entity.action)} dot={false}>
                                            {actionLabel(entity.action)}
                                        </StatusBadge>
                                    </TableCell>
                                </TableRow>
                            ))
                          : segment === "groups"
                            ? groups.map((group, at) => {
                                  const trail = trailOf(group.path ?? []);
                                  const page = group.page ?? "";
                                  return (
                                      <TableRow key={`${group.sheet ?? ""}-${trail}-${at}`}>
                                          <SheetCell shown={shown} sheet={group.sheet} />
                                          <TableCell title={trail}>{trail}</TableCell>
                                          <TableCell mono={page !== ""} muted={page === ""} title={page}>
                                              {page === "" ? copy.imports.preview.noPage : page}
                                          </TableCell>
                                          <TableCell mono={true} muted={true}>
                                              {group.rows}
                                          </TableCell>
                                      </TableRow>
                                  );
                              })
                            : edges.map((edge, at) => (
                                  <TableRow key={`${edge.sheet ?? ""}-${edge.from}-${edge.to}-${at}`}>
                                      <SheetCell shown={shown} sheet={edge.sheet} />
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
                        disabled={errors.length > 0}
                        title={errors.length > 0 ? copy.imports.preview.blocked : undefined}
                        onClick={onNext}
                    >
                        {copy.imports.next}
                    </Button>
                </div>
            </div>
        </div>
    );
}
