import type { ReactElement } from "react";
import { useMemo, useState } from "react";

import { copy } from "../../copy/index.js";
import type { PreviewReport } from "../../data/types.js";
import { importActions } from "../../generated/vocab.js";
import { Banner, Button, DenseTable, Segmented, TableHead, cx, toneClasses } from "../../ui/index.js";
import type { Choices } from "./choices.js";
import { actionLabel, actionTone } from "./labels.js";
import { sheetsIn } from "./preview.js";
import { ColumnUses } from "./preview-columns.js";
import { EdgeRows, EntityRows, GroupRows, PageRows } from "./preview-rows.js";

type Segment = "pages" | "entities" | "groups" | "edges";

const grids: Readonly<Record<Segment, string>> = {
    pages: "minmax(0,1.8fr) minmax(0,1.4fr) minmax(0,1.1fr) minmax(0,0.9fr) minmax(0,0.8fr) minmax(0,1.6fr) 6rem",
    entities: "minmax(0,1.8fr) minmax(0,1.4fr) 6rem",
    groups: "minmax(0,2fr) minmax(0,1.6fr) 5rem",
    edges: "minmax(0,1.4fr) minmax(0,1.4fr) 7rem 6rem",
};

const sheetTrack = "minmax(0,0.8fr)";

const said = copy.imports.preview;

const headers: Readonly<Record<Segment, readonly string[]>> = {
    pages: [said.columnPath, said.columnTitle, said.columnKeyword, said.columnEntity, said.columnType, said.columnMatched, said.columnAction],
    entities: [said.columnName, said.columnKeyword, said.columnAction],
    groups: [said.columnGroup, said.columnPage, said.columnRows],
    edges: [said.columnFrom, said.columnTo, said.columnRelation, said.columnAction],
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
    choices: Choices;
    onPick: (rowKey: string, pageId: string) => void;
    onBack: () => void;
    onNext: () => void;
}

export function StepPreview({ report, choices, onPick, onBack, onNext }: StepPreviewProps): ReactElement {
    const [segment, setSegment] = useState<Segment>("pages");
    const pages = useMemo(() => report.pages ?? [], [report.pages]);
    const entities = useMemo(() => report.entities ?? [], [report.entities]);
    const groups = useMemo(() => report.groups ?? [], [report.groups]);
    const edges = useMemo(() => report.edges ?? [], [report.edges]);
    const shown = useMemo(() => sheetsIn(report).length > 1, [report]);
    const errors = report.errors ?? [];
    const counted = segment === "pages" ? tally(pages) : segment === "entities" ? tally(entities) : segment === "edges" ? tally(edges) : [];

    return (
        <div className="flex min-h-0 flex-1 flex-col">
            <div className="flex shrink-0 flex-wrap items-center gap-2 border-b border-hairline px-3 py-2">
                <Segmented
                    label={said.title}
                    value={segment}
                    options={[
                        { value: "pages", label: `${said.pages} ${pages.length}` },
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
                    {report.skipped === 0 ? null : <span className="text-2xs text-ink-faint">{said.skipped(report.skipped)}</span>}
                </div>
            </div>
            <ColumnUses columns={report.columns ?? []} />
            <div className="min-h-0 flex-1 overflow-auto">
                <DenseTable columns={shown ? `${sheetTrack} ${grids[segment]}` : grids[segment]} label={said.title}>
                    <TableHead>
                        {shown ? <span>{said.columnSheet}</span> : null}
                        {headers[segment].map((header) => (
                            <span key={header}>{header}</span>
                        ))}
                    </TableHead>
                    {segment === "pages" ? <PageRows pages={pages} shown={shown} choices={choices} onPick={onPick} /> : null}
                    {segment === "entities" ? <EntityRows entities={entities} shown={shown} /> : null}
                    {segment === "groups" ? <GroupRows groups={groups} shown={shown} /> : null}
                    {segment === "edges" ? <EdgeRows edges={edges} shown={shown} /> : null}
                </DenseTable>
            </div>
            <div className="flex shrink-0 items-center gap-3 border-t border-hairline p-3">
                {errors.length === 0 ? null : <Banner tone="danger" title={said.blocked} className="min-w-0 flex-1" />}
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
