import type { ReactElement, ReactNode } from "react";
import { useCallback, useMemo } from "react";

import { copy } from "../../../copy/index.js";
import { pushToast } from "../../../data/toasts.js";
import type { SiteMapNode } from "../../../data/types.js";
import type { Selection, SelectionRow } from "../../../ui/selection.js";
import { clearSubtree, range, selectSubtree, stateOf } from "../../../ui/selection.js";
import type { MapEntry, MapScope, MapSelection } from "../../map/tree.js";
import { MapTree } from "../../map/tree.js";
import type { KnownRow } from "./model.js";
import { childrenFrom, nodeRow, nodeSize } from "./model.js";
import type { TypeNaming } from "./naming.js";

export interface PagesTreeProps {
    siteId: string;
    selection: Selection;
    onSelection: (next: Selection) => void;
    naming: TypeNaming;
    within?: string;
    scrollKey: string;
    expanded: ReadonlySet<string>;
    onExpanded: (next: ReadonlySet<string>) => void;
    empty: ReactNode;
    onRows: (rows: ReadonlyMap<string, KnownRow>) => void;
    onOpen: (pageId: string) => void;
    onUnder: (pageId: string) => void;
}

export function PagesTree({
    siteId,
    selection,
    onSelection,
    naming,
    within,
    scrollKey,
    expanded,
    onExpanded,
    empty,
    onRows,
    onOpen,
    onUnder,
}: PagesTreeProps): ReactElement {
    const { typeIdOf } = naming;
    const rowOf = useCallback((node: SiteMapNode, ancestors: readonly string[]): SelectionRow => nodeRow(node, ancestors, typeIdOf), [typeIdOf]);

    const tick = useMemo<MapSelection>(
        () => ({
            label: (node) => copy.pages.select.branch(node.page.path),
            stateOf: (node, ancestors) =>
                stateOf(selection, node.page.pageId, rowOf(node, ancestors), within === undefined ? { leaf: node.children === 0 } : { leaf: node.children === 0, typeId: within }),
            onTick: (node: SiteMapNode, ancestors: readonly string[], ranged: boolean, scope: MapScope) => {
                const id = node.page.pageId;
                const row = rowOf(node, ancestors);
                const chains = new Map(scope.visible.map((entry) => [entry.id, entry.ancestors] as const));
                const childrenOf = childrenFrom(scope.branches, (pageId) => chains.get(pageId), typeIdOf, within);
                if (ranged) {
                    const shown = scope.visible.filter((entry) => within === undefined || typeIdOf(entry.node.page.type) === within);
                    const rows = new Map(shown.map((entry) => [entry.id, rowOf(entry.node, entry.ancestors)] as const));
                    onSelection(range(selection, shown.map((entry) => entry.id), id, (pageId) => rows.get(pageId), childrenOf));
                    return;
                }
                const pick = within === undefined ? { pageId: id, size: nodeSize(node) } : { pageId: id, typeId: within, size: null };
                const scopeOf = within === undefined ? { leaf: node.children === 0 } : { leaf: node.children === 0, typeId: within };
                if (stateOf(selection, id, row, scopeOf) === "on") {
                    onSelection(clearSubtree(selection, pick, row));
                    return;
                }
                const next = selectSubtree(selection, pick, row, childrenOf);
                if (next === null) {
                    pushToast("warning", copy.pages.select.refused);
                    return;
                }
                onSelection(next);
            },
        }),
        [selection, onSelection, rowOf, typeIdOf, within],
    );

    const visible = useCallback(
        (entries: readonly MapEntry[]) => {
            onRows(new Map(entries.map((entry) => [entry.id, { path: entry.node.page.path, row: rowOf(entry.node, entry.ancestors) }] as const)));
        },
        [onRows, rowOf],
    );

    return (
        <MapTree
            key={`${siteId}:${within ?? ""}`}
            siteId={siteId}
            root=""
            within={within}
            selection={tick}
            scrollKey={scrollKey}
            expanded={expanded}
            onExpanded={onExpanded}
            onVisible={visible}
            empty={empty}
            onOpen={onOpen}
            onUnder={onUnder}
        />
    );
}
