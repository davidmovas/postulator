import type { ReactElement } from "react";
import { useMemo, useState } from "react";

import { copy } from "../../copy/index.js";
import type { PreviewReport } from "../../data/types.js";
import { categoryPathKey, plannedItems } from "../../domain/categories.js";
import { trailOf } from "../../domain/entities.js";
import { keywordsLine } from "../../domain/keywords.js";
import { importActions, importCategoryActions } from "../../generated/vocab.js";
import type { Tone } from "../../ui/index.js";
import {
    Banner,
    Button,
    CategoryTrail,
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
import {
    actionLabel,
    actionTone,
    categoryActionLabel,
    categoryActionTone,
    columnUseLabel,
    edgeKindLabel,
    productNote,
} from "./labels.js";
import type { PreviewColumn } from "./preview.js";
import { columnsBySheet, createdPaths, sheetsIn } from "./preview.js";

type Segment = "pages" | "categories" | "entities" | "groups" | "edges";

const grids: Readonly<Record<Segment, string>> = {
    pages: "minmax(0,1.8fr) minmax(0,1.4fr) minmax(0,1.1fr) minmax(0,0.9fr) minmax(0,1.3fr) 6rem",
    categories: "minmax(0,2.6fr) 5rem 6rem",
    entities: "minmax(0,1.8fr) 7rem minmax(0,1.4fr) 6rem",
    groups: "minmax(0,2fr) minmax(0,1.6fr) 5rem",
    edges: "minmax(0,1.4fr) minmax(0,1.4fr) 7rem 6rem",
};

const sheetTrack = "minmax(0,0.8fr)";

interface Countable {
    action: string;
}

interface Tally {
    action: string;
    count: number;
    label: string;
    tone: Tone;
}

function tally(rows: readonly Countable[]): Tally[] {
    return importActions
        .map((action) => ({
            action,
            count: rows.filter((row) => row.action === action).length,
            label: actionLabel(action),
            tone: actionTone(action),
        }))
        .filter((held) => held.count > 0);
}

function tallyCategories(rows: readonly Countable[]): Tally[] {
    return importCategoryActions
        .map((action) => ({
            action,
            count: rows.filter((row) => row.action === action).length,
            label: categoryActionLabel(action),
            tone: categoryActionTone(action),
        }))
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

function Head({ segment, shown }: { segment: Segment; shown: boolean }): ReactElement {
    const said = copy.imports.preview;
    return (
        <TableHead>
            {shown ? <span>{said.columnSheet}</span> : null}
            {segment === "pages" ? (
                <>
                    <span>{said.columnPath}</span>
                    <span>{said.columnTitle}</span>
                    <span>{said.columnKeyword}</span>
                    <span>{said.columnEntity}</span>
                    <span>{said.columnCategories}</span>
                    <span>{said.columnAction}</span>
                </>
            ) : segment === "categories" ? (
                <>
                    <span>{said.columnCategory}</span>
                    <span>{said.columnRows}</span>
                    <span>{said.columnAction}</span>
                </>
            ) : segment === "entities" ? (
                <>
                    <span>{said.columnName}</span>
                    <span>{said.columnKind}</span>
                    <span>{said.columnKeyword}</span>
                    <span>{said.columnAction}</span>
                </>
            ) : segment === "groups" ? (
                <>
                    <span>{said.columnGroup}</span>
                    <span>{said.columnPage}</span>
                    <span>{said.columnRows}</span>
                </>
            ) : (
                <>
                    <span>{said.columnFrom}</span>
                    <span>{said.columnTo}</span>
                    <span>{said.columnRelation}</span>
                    <span>{said.columnAction}</span>
                </>
            )}
        </TableHead>
    );
}

interface RowsProps {
    report: PreviewReport;
    shown: boolean;
    created: ReadonlySet<string>;
}

function PageRows({ report, shown, created }: RowsProps): ReactElement {
    return (
        <>
            {(report.pages ?? []).map((page, at) => (
                <TableRow key={`${page.sheet ?? ""}-${page.path}-${at}`}>
                    <SheetCell shown={shown} sheet={page.sheet} />
                    <TableCell mono={true} title={page.path}>
                        {page.path}
                    </TableCell>
                    <TableCell muted={productNote(page) !== null} title={productNote(page) ?? page.title}>
                        {productNote(page) ?? page.title}
                    </TableCell>
                    <TableCell muted={true} title={keywordsLine(page.keywords)}>
                        {keywordsLine(page.keywords)}
                    </TableCell>
                    <TableCell muted={true}>{page.entity ?? ""}</TableCell>
                    <TableCell>
                        <CategoryTrail
                            items={plannedItems(page.categories ?? [], created)}
                            label={copy.categories.trail}
                            compact={true}
                        />
                    </TableCell>
                    <TableCell>
                        <StatusBadge tone={actionTone(page.action)} dot={false}>
                            {actionLabel(page.action)}
                        </StatusBadge>
                    </TableCell>
                </TableRow>
            ))}
        </>
    );
}

function CategoryRows({ report, shown, created }: RowsProps): ReactElement {
    return (
        <>
            {(report.categories ?? []).map((category, at) => {
                const path = category.path ?? [];
                return (
                    <TableRow
                        key={`${category.sheet ?? ""}-${categoryPathKey(path)}-${category.action}-${at}`}
                        data-preview-category={category.action}
                    >
                        <SheetCell shown={shown} sheet={category.sheet} />
                        <TableCell>
                            <CategoryTrail items={plannedItems(path, created)} label={copy.categories.trail} compact={true} />
                        </TableCell>
                        <TableCell mono={true} muted={true}>
                            {category.rows > 0 ? category.rows : ""}
                        </TableCell>
                        <TableCell>
                            <StatusBadge tone={categoryActionTone(category.action)} dot={false}>
                                {categoryActionLabel(category.action)}
                            </StatusBadge>
                        </TableCell>
                    </TableRow>
                );
            })}
        </>
    );
}

function EntityRows({ report, shown }: RowsProps): ReactElement {
    return (
        <>
            {(report.entities ?? []).map((entity, at) => (
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
            ))}
        </>
    );
}

function GroupRows({ report, shown }: RowsProps): ReactElement {
    return (
        <>
            {(report.groups ?? []).map((group, at) => {
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
            })}
        </>
    );
}

function EdgeRows({ report, shown }: RowsProps): ReactElement {
    return (
        <>
            {(report.edges ?? []).map((edge, at) => (
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
        </>
    );
}

const segmentRows: Readonly<Record<Segment, (props: RowsProps) => ReactElement>> = {
    pages: PageRows,
    categories: CategoryRows,
    entities: EntityRows,
    groups: GroupRows,
    edges: EdgeRows,
};

export interface StepPreviewProps {
    report: PreviewReport;
    onBack: () => void;
    onNext: () => void;
}

export function StepPreview({ report, onBack, onNext }: StepPreviewProps): ReactElement {
    const [segment, setSegment] = useState<Segment>("pages");
    const pages = report.pages ?? [];
    const categories = report.categories ?? [];
    const entities = report.entities ?? [];
    const groups = report.groups ?? [];
    const edges = report.edges ?? [];
    const shown = useMemo(() => sheetsIn(report).length > 1, [report]);
    const created = useMemo(() => createdPaths(report), [report]);
    const errors = report.errors ?? [];
    const said = copy.imports.preview;
    const counted = ((): Tally[] => {
        switch (segment) {
            case "pages":
                return tally(pages);
            case "categories":
                return tallyCategories(categories);
            case "entities":
                return tally(entities);
            case "edges":
                return tally(edges);
            default:
                return [];
        }
    })();
    const Rows = segmentRows[segment];

    return (
        <div className="flex min-h-0 flex-1 flex-col">
            <div className="flex shrink-0 flex-wrap items-center gap-2 border-b border-hairline px-3 py-2">
                <Segmented
                    label={said.title}
                    value={segment}
                    options={[
                        { value: "pages", label: `${said.pages} ${pages.length}` },
                        { value: "categories", label: `${said.categories} ${categories.length}` },
                        { value: "entities", label: `${said.entities} ${entities.length}` },
                        { value: "groups", label: `${said.groups} ${groups.length}` },
                        { value: "edges", label: `${said.edges} ${edges.length}` },
                    ]}
                    onValueChange={setSegment}
                />
                <div className="flex flex-wrap items-center gap-1.5">
                    {counted.map((held) => (
                        <span
                            key={held.action}
                            data-tally={held.action}
                            className={cx(
                                "flex h-5 items-center gap-1 rounded-sm px-1.5 text-2xs",
                                toneClasses[held.tone].soft,
                                toneClasses[held.tone].ink,
                            )}
                        >
                            <span className="font-mono">{held.count}</span>
                            {held.label}
                        </span>
                    ))}
                    {report.skipped === 0 ? null : (
                        <span className="text-2xs text-ink-faint">{said.skipped(report.skipped)}</span>
                    )}
                </div>
            </div>
            <ColumnUses columns={report.columns ?? []} />
            {segment === "categories" ? (
                <p className="shrink-0 border-b border-hairline px-3 py-1.5 text-2xs text-ink-faint">
                    {categories.length === 0 ? said.noCategories : said.categoriesHint}
                </p>
            ) : null}
            <div className="min-h-0 flex-1 overflow-auto">
                <DenseTable columns={shown ? `${sheetTrack} ${grids[segment]}` : grids[segment]} label={said.title}>
                    <Head segment={segment} shown={shown} />
                    <Rows report={report} shown={shown} created={created} />
                </DenseTable>
            </div>
            <div className="flex shrink-0 items-center gap-3 border-t border-hairline p-3">
                {errors.length === 0 ? null : (
                    <Banner tone="danger" title={said.blocked} className="min-w-0 flex-1" />
                )}
                <div className="ml-auto flex shrink-0 gap-2">
                    <Button variant="secondary" onClick={onBack}>
                        {copy.imports.back}
                    </Button>
                    <Button
                        variant="primary"
                        disabled={errors.length > 0}
                        title={errors.length > 0 ? said.blocked : undefined}
                        onClick={onNext}
                    >
                        {copy.imports.next}
                    </Button>
                </div>
            </div>
        </div>
    );
}
