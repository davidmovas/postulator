import type { ReactElement, ReactNode } from "react";
import { useCallback, useMemo, useState } from "react";

import { copy } from "../../../copy/index.js";
import { errorMessageOf } from "../../../data/errors.js";
import type { SiteMapNode } from "../../../data/types.js";
import { Button, Input, SectionLabel, Skeleton, SkeletonRows, VirtualRows, useDebounced } from "../../../ui/index.js";
import type { Loaded } from "../../map/loader.js";
import { BranchLoader, useBranch } from "../../map/loader.js";
import { flattenTree, openBranches } from "../../map/model/tree.js";
import type { Branches, TreeRow } from "../../map/model/tree.js";
import type { Keep, NamedPage } from "./model.js";
import { everyNode, kept, labelOf, tick, toggled } from "./model.js";
import { PageRow } from "./row.js";

const searchDelayMs = 250;
const rootParent = "";
const rowHeight = 28;
const minHeight = 128;
const maxHeight = 288;
const framing = 8;
const indentPx = 14;

export interface PageTreeProps {
    siteId: string;
    selected: ReadonlySet<string>;
    onChange: (next: ReadonlySet<string>, named: readonly NamedPage[]) => void;
    pickable: Keep;
    keep?: Keep;
    required?: ReadonlyMap<string, string>;
    label: string;
    picked: string;
    refusalOf?: (node: SiteMapNode) => string | null;
    noteOf?: (node: SiteMapNode) => string | null;
    filter?: ReactNode;
    problem?: string | null;
}

const noRequired: ReadonlyMap<string, string> = new Map();

function foundRows(nodes: readonly SiteMapNode[], keep: Keep, more: boolean): TreeRow[] {
    const rows: TreeRow[] = nodes.filter(keep).map((node) => ({ kind: "node", key: `found:${node.page.pageId}`, node, depth: 0, expanded: false }));
    return more ? [...rows, { kind: "more", key: "more:found", parentId: rootParent, depth: 0 }] : rows;
}

export function PageTree({
    siteId,
    selected,
    onChange,
    pickable,
    keep = everyNode,
    required = noRequired,
    label,
    picked,
    refusalOf,
    noteOf,
    filter,
    problem,
}: PageTreeProps): ReactElement {
    const [typed, setTyped] = useState("");
    const search = useDebounced(typed.trim(), searchDelayMs);
    const searching = search !== "";
    const [expanded, setExpanded] = useState<ReadonlySet<string>>(new Set());
    const [loaded, setLoaded] = useState<ReadonlyMap<string, Loaded>>(new Map());
    const found = useBranch({ siteId, search }, searching);

    const onLoaded = useCallback((parentId: string, held: Loaded) => {
        setLoaded((prev) => (prev.get(parentId) === held ? prev : new Map(prev).set(parentId, held)));
    }, []);

    const branches = useMemo<Branches>(() => new Map([...loaded].map(([key, held]) => [key, held.branch])), [loaded]);
    const open = useMemo(() => openBranches(rootParent, expanded, branches), [expanded, branches]);
    const rows = useMemo(
        () =>
            searching
                ? foundRows(found.branch.nodes, keep, found.branch.hasMore)
                : flattenTree(rootParent, expanded, branches).filter((row) => row.kind !== "node" || kept(row.node, keep)),
        [searching, found.branch, keep, expanded, branches],
    );
    const top = searching ? found : loaded.get(rootParent);
    const pending = top === undefined || (top.branch.pending && top.branch.nodes.length === 0);
    const failed = top === undefined ? null : errorMessageOf(top.error);
    const narrowed = searching || keep !== everyNode;

    const renderRow = (row: TreeRow): ReactNode => {
        if (row.kind === "more") {
            return (
                <div style={{ paddingLeft: row.depth * indentPx }}>
                    <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => {
                            (searching ? found : loaded.get(row.parentId))?.more();
                        }}
                    >
                        {copy.pages.pick.more}
                    </Button>
                </div>
            );
        }
        if (row.kind === "loading") {
            return (
                <div aria-label={copy.pages.loading} className="flex h-7 items-center" style={{ paddingLeft: (row.depth + 1) * indentPx }}>
                    <Skeleton width="40%" height={10} />
                </div>
            );
        }
        const node = row.node;
        const id = node.page.pageId;
        const refusal = refusalOf?.(node) ?? (pickable(node) ? null : copy.pages.pick.unavailable);
        return (
            <PageRow
                node={node}
                label={labelOf(node, searching)}
                depth={row.depth}
                expanded={searching || node.children === 0 ? null : row.expanded}
                tick={tick(id, selected, required, branches)}
                refusal={refusal}
                note={noteOf?.(node) ?? null}
                onToggle={() => {
                    onChange(toggled(selected, [id], !selected.has(id)), [{ id, path: node.page.path }]);
                }}
                onExpand={() => {
                    setExpanded((held) => toggled(held, [id], !held.has(id)));
                }}
            />
        );
    };

    return (
        <div className="flex flex-col gap-1.5">
            {open.map((parentId) => (
                <BranchLoader key={parentId} siteId={siteId} parentId={parentId} onLoaded={onLoaded} />
            ))}
            <div className="flex items-center justify-between gap-2">
                <SectionLabel>{label}</SectionLabel>
                <span className="font-mono text-2xs text-ink-faint">{picked}</span>
            </div>
            <div className="flex items-center gap-2">
                <div className="min-w-0 flex-1">
                    <Input
                        type="search"
                        value={typed}
                        placeholder={copy.pages.pick.searchHint}
                        aria-label={copy.pages.pick.search}
                        onChange={(event) => {
                            setTyped(event.target.value);
                        }}
                    />
                </div>
                {filter}
            </div>
            <div
                className="flex flex-col overflow-hidden rounded-md border border-hairline bg-inset p-1"
                style={{ height: Math.min(maxHeight, Math.max(minHeight, rows.length * rowHeight + framing)) }}
            >
                {failed !== null ? (
                    <p className="p-1 text-xs text-danger">{failed}</p>
                ) : pending ? (
                    <SkeletonRows rows={4} label={copy.pages.loading} />
                ) : rows.length === 0 ? (
                    <p className="p-1 text-xs text-ink-dim">{narrowed ? copy.pages.pick.noMatch : copy.empty.pages}</p>
                ) : (
                    <VirtualRows
                        count={rows.length}
                        rowHeight={rowHeight}
                        row={(at) => {
                            const row = rows[at];
                            return row === undefined ? null : renderRow(row);
                        }}
                    />
                )}
            </div>
            {typeof problem === "string" && problem !== "" ? (
                <p data-run-targets-problem={true} role="alert" className="text-xs text-danger">
                    {problem}
                </p>
            ) : null}
        </div>
    );
}
