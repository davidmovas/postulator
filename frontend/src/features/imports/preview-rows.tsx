import type { ReactElement } from "react";

import { copy } from "../../copy/index.js";
import type { PreviewEdge, PreviewEntity, PreviewGroup, PreviewPage } from "../../data/types.js";
import { trailOf } from "../../domain/entities.js";
import { keywordsLine } from "../../domain/keywords.js";
import { StatusBadge, TableCell, TableRow } from "../../ui/index.js";
import type { Choices } from "./choices.js";
import { actionLabel, actionTone, edgeKindLabel, productNote } from "./labels.js";
import { MatchCell } from "./match-cell.js";

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

function ActionCell({ action }: { action: string }): ReactElement {
    return (
        <TableCell>
            <StatusBadge tone={actionTone(action)} dot={false}>
                {actionLabel(action)}
            </StatusBadge>
        </TableCell>
    );
}

export interface PageRowsProps {
    pages: readonly PreviewPage[];
    shown: boolean;
    choices: Choices;
    onPick: (rowKey: string, pageId: string) => void;
}

export function PageRows({ pages, shown, choices, onPick }: PageRowsProps): ReactElement {
    return (
        <>
            {pages.map((page, at) => (
                <TableRow key={`${page.sheet ?? ""}-${page.rowKey ?? page.path}-${at}`}>
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
                    <TableCell muted={true}>{page.pageType ?? ""}</TableCell>
                    <MatchCell page={page} choices={choices} onPick={onPick} />
                    <ActionCell action={page.action} />
                </TableRow>
            ))}
        </>
    );
}

export function EntityRows({ entities, shown }: { entities: readonly PreviewEntity[]; shown: boolean }): ReactElement {
    return (
        <>
            {entities.map((entity, at) => (
                <TableRow key={`${entity.sheet ?? ""}-${entity.name}-${at}`}>
                    <SheetCell shown={shown} sheet={entity.sheet} />
                    <TableCell title={trailOf([entity.parent ?? "", entity.name])}>
                        {trailOf([entity.parent ?? "", entity.name])}
                    </TableCell>
                    <TableCell mono={true} muted={true} title={keywordsLine(entity.keywords)}>
                        {keywordsLine(entity.keywords)}
                    </TableCell>
                    <ActionCell action={entity.action} />
                </TableRow>
            ))}
        </>
    );
}

export function GroupRows({ groups, shown }: { groups: readonly PreviewGroup[]; shown: boolean }): ReactElement {
    return (
        <>
            {groups.map((group, at) => {
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

export function EdgeRows({ edges, shown }: { edges: readonly PreviewEdge[]; shown: boolean }): ReactElement {
    return (
        <>
            {edges.map((edge, at) => (
                <TableRow key={`${edge.sheet ?? ""}-${edge.from}-${edge.to}-${at}`}>
                    <SheetCell shown={shown} sheet={edge.sheet} />
                    <TableCell title={edge.from}>{edge.from}</TableCell>
                    <TableCell title={edge.to}>{edge.to}</TableCell>
                    <TableCell muted={true}>{edgeKindLabel(edge.kind)}</TableCell>
                    <ActionCell action={edge.action} />
                </TableRow>
            ))}
        </>
    );
}
