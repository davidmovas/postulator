import type { ReactElement } from "react";

import { copy } from "../../../copy/index.js";
import type { ProposedEntity } from "../../../data/types.js";
import { keywordList, mainKeyword } from "../../../domain/keywords.js";
import { Button, Checkbox, DenseTable, TableCell, TableHead, TableRow, VirtualRows } from "../../../ui/index.js";
import { clashLabel } from "../labels.js";
import { everyTickable, joinsExisting, placeOf, tickable, toggled } from "../model/proposals.js";

const said = copy.graph.ai;

const modelColumns = "3rem minmax(0,1.6fr) minmax(0,1.4fr) minmax(0,1.4fr)";
const siteColumns = "3rem minmax(0,1.8fr) minmax(0,1.3fr) minmax(0,1.1fr)";
const modelRowHeight = 28;
const siteRowHeight = 44;

interface RowProps {
    proposal: ProposedEntity;
    at: number;
    kept: boolean;
    onToggle: (at: number) => void;
}

function ModelRow({ proposal, at, kept, onToggle }: RowProps): ReactElement {
    const keyword = mainKeyword(keywordList(proposal.keywords));
    const place = placeOf(proposal);
    return (
        <TableRow data-proposal={proposal.name} data-proposal-page={proposal.pageId}>
            <TableCell>
                <Checkbox checked={kept} aria-label={proposal.name} onChange={() => onToggle(at)} />
            </TableCell>
            <TableCell title={proposal.name}>
                {proposal.name}
                {joinsExisting(proposal) ? <span className="ml-1 text-2xs text-ink-faint">{said.existing}</span> : null}
            </TableCell>
            <TableCell muted={true} title={keyword}>
                {keyword}
            </TableCell>
            <TableCell mono={true} muted={true} title={place}>
                {place}
            </TableCell>
        </TableRow>
    );
}

function SiteRow({ proposal, at, kept, onToggle }: RowProps): ReactElement {
    const label = (proposal.label ?? "") === "" ? proposal.name : (proposal.label ?? "");
    const path = proposal.path ?? "";
    const parent = proposal.parent ?? "";
    const clash = proposal.clash ?? "";
    const facts = proposal.facts ?? [];
    const factLine = facts.length === 0 ? "" : said.facts(facts);
    const allowed = tickable(proposal);
    return (
        <TableRow data-proposal={proposal.name} data-proposal-page={proposal.pageId} style={{ height: siteRowHeight }}>
            <TableCell>
                <Checkbox
                    checked={allowed && kept}
                    disabled={!allowed}
                    aria-label={said.keepRow(label === "" ? said.unnamed : label, path)}
                    onChange={() => onToggle(at)}
                />
            </TableCell>
            <div role="cell" className="flex min-w-0 flex-col justify-center gap-0.5">
                <span className={label === "" ? "truncate text-ink-faint" : "truncate text-ink"} title={label}>
                    {label === "" ? said.unnamed : label}
                </span>
                <span className="flex min-w-0 items-center gap-2 text-2xs">
                    {clash === "" ? null : <span className="truncate text-warn">{clashLabel(clash)}</span>}
                    {clash === "" && joinsExisting(proposal) ? <span className="shrink-0 text-ink-faint">{said.joins}</span> : null}
                    {factLine === "" ? null : (
                        <span className="truncate text-ink-dim" title={factLine}>
                            {factLine}
                        </span>
                    )}
                </span>
            </div>
            <TableCell mono={true} muted={true} title={path}>
                {path}
            </TableCell>
            <TableCell mono={true} muted={true} title={parent === "" ? said.topLevel : parent}>
                {parent === "" ? said.topLevel : parent}
            </TableCell>
        </TableRow>
    );
}

export interface ProposalReviewProps {
    proposals: readonly ProposedEntity[];
    kept: ReadonlySet<number>;
    onKept: (next: ReadonlySet<number>) => void;
    fromSite: boolean;
}

export function ProposalReview({ proposals, kept, onKept, fromSite }: ProposalReviewProps): ReactElement {
    const toggle = (at: number): void => {
        onKept(toggled(proposals, kept, at));
    };
    const Row = fromSite ? SiteRow : ModelRow;
    return (
        <>
            <div className="flex shrink-0 items-center justify-between gap-2">
                <p className="text-xs text-ink-soft">
                    <span className="font-medium text-ink">{said.reviewTitle(proposals.length)}</span>{" "}
                    {fromSite ? said.siteReviewBody : said.reviewBody}
                </p>
                <div className="flex shrink-0 gap-1">
                    <Button size="sm" variant="ghost" onClick={() => onKept(everyTickable(proposals))}>
                        {said.keepAll}
                    </Button>
                    <Button size="sm" variant="ghost" onClick={() => onKept(new Set())}>
                        {said.keepNone}
                    </Button>
                </div>
            </div>
            <DenseTable
                columns={fromSite ? siteColumns : modelColumns}
                label={said.reviewTitle(proposals.length)}
                className="flex min-h-0 flex-1 flex-col"
            >
                <TableHead>
                    <span>{said.columnKeep}</span>
                    <span>{said.columnName}</span>
                    <span>{fromSite ? said.columnPage : said.columnKeyword}</span>
                    <span>{fromSite ? said.columnParent : said.columnPlace}</span>
                </TableHead>
                <VirtualRows
                    count={proposals.length}
                    rowHeight={fromSite ? siteRowHeight : modelRowHeight}
                    row={(at) => {
                        const proposal = proposals[at];
                        return proposal === undefined ? null : <Row proposal={proposal} at={at} kept={kept.has(at)} onToggle={toggle} />;
                    }}
                />
            </DenseTable>
        </>
    );
}

export interface ProposalConfirmProps {
    written: number;
    joined: number;
}

export function ProposalConfirm({ written, joined }: ProposalConfirmProps): ReactElement {
    return (
        <div className="flex flex-col gap-1.5 rounded-lg border border-hairline bg-inset p-3" data-proposal-confirm={true}>
            <span className="text-sm font-semibold text-ink">{said.confirmTitle}</span>
            <p className="text-xs leading-snug text-ink-soft">{said.confirmBody(written, joined)}</p>
        </div>
    );
}
