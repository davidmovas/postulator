import type { ReactElement, ReactNode } from "react";
import { useCallback, useMemo, useState } from "react";

import { copy } from "../../../copy/index.js";
import { errorMessageOf } from "../../../data/errors.js";
import type { SiteMapNode } from "../../../data/types.js";
import { Button, Input, SectionLabel, Skeleton, SkeletonRows, VirtualRows, cx, useDebounced } from "../../../ui/index.js";
import type { Loaded } from "../../map/loader.js";
import { BranchLoader, useBranch } from "../../map/loader.js";
import { flattenTree, openBranches } from "../../map/model/tree.js";
import type { Branches, TreeRow } from "../../map/model/tree.js";
import type { Keep, NamedPage } from "./model.js";
import { everyNode, kept, labelOf, nextPick, tick, toggled } from "./model.js";
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
    view?: TreeView;
    onView?: (next: TreeView) => void;
    fill?: boolean;
}

export interface TreeView {
    search: string;
    expanded: ReadonlySet<string>;
}

export const blankTreeView: TreeView = Object.freeze({ search: "", expanded: new Set<string>() });

const noRequired: ReadonlyMap<string, string> = new Map();

function foundRows(nodes: readonly SiteMapNode[], keep: Keep, more: boolean): TreeRow[] {
    const rows: TreeRow[] = nodes
        .filter(keep)
        .map((node) => ({ kind: "node", key: `found:${node.page.pageId}`, node, depth: 0, expanded: false, ancestors: [] }));
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
    view,
    onView,
    fill = false,
}: PageTreeProps): ReactElement {
    const [own, setOwn] = useState<TreeView>(blankTreeView);
    const held = view ?? own;
    const setView = onView ?? setOwn;
    const typed = held.search;
    const expanded = held.expanded;
    const search = useDebounced(typed.trim(), searchDelayMs);
    const searching = search !== "";
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
    const [anchor, setAnchor] = useState<string | null>(null);
    const top = searching ? found : loaded.get(rootParent);
    const pending = top === undefined || (top.branch.pending && top.branch.nodes.length === 0);
    const failed = top === undefined ? null : errorMessageOf(top.error);
    const narrowed = searching || keep !== everyNode;

    const refusalFor = (node: SiteMapNode): string | null => refusalOf?.(node) ?? (pickable(node) ? null : copy.pages.pick.unavailable);
    const shown = (): NamedPage[] =>
        rows.flatMap((row) =>
            row.kind === "node" && refusalFor(row.node) === null ? [{ id: row.node.page.pageId, path: row.node.page.path }] : [],
        );

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
        const refusal = refusalFor(node);
        return (
            <PageRow
                node={node}
                label={labelOf(node, searching)}
                depth={row.depth}
                expanded={searching || node.children === 0 ? null : row.expanded}
                tick={tick(id, selected, required, branches)}
                refusal={refusal}
                note={noteOf?.(node) ?? null}
                onToggle={(ranged) => {
                    const next = nextPick(selected, shown(), anchor, { id, path: node.page.path }, ranged);
                    setAnchor(next.anchor);
                    onChange(next.selected, next.named);
                }}
                onExpand={() => {
                    setView({ search: typed, expanded: toggled(expanded, [id], !expanded.has(id)) });
                }}
            />
        );
    };

    return (
        <div className={cx("flex flex-col gap-1.5", fill && "min-h-0 flex-1")}>
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
                            setView({ search: event.target.value, expanded });
                        }}
                    />
                </div>
                {filter}
            </div>
            <div
                className={cx("flex flex-col overflow-hidden rounded-md border border-hairline bg-inset p-1", fill && "min-h-0 flex-1")}
                style={fill ? undefined : { height: Math.min(maxHeight, Math.max(minHeight, rows.length * rowHeight + framing)) }}
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
