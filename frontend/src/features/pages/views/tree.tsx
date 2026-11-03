import type { ReactElement, ReactNode } from "react";
import { useCallback, useMemo } from "react";

import { copy } from "../../../copy/index.js";
import { pushToast } from "../../../data/toasts.js";
import type { AddressNode } from "../../../data/types.js";
import type { Selection, SelectionRow } from "../../../ui/selection.js";
import { clearSubtree, range, selectSubtree, stateOf } from "../../../ui/selection.js";
import { addressChildren, addressRow, keyOf } from "../address/model.js";
import type { AddressEntry, AddressScope, AddressSelection } from "../address/tree.js";
import { AddressTree } from "../address/tree.js";
import type { KnownRow } from "./model.js";
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
    onUnder: (path: string) => void;
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
    const rowOf = useCallback((node: AddressNode, ancestors: readonly string[]): SelectionRow => addressRow(node, ancestors, typeIdOf), [typeIdOf]);

    const tick = useMemo<AddressSelection>(
        () => ({
            label: (node) => copy.pages.select.branch(node.path),
            stateOf: (node, ancestors) =>
                stateOf(selection, keyOf(node), rowOf(node, ancestors), within === undefined ? { leaf: node.children === 0 } : { leaf: node.children === 0, typeId: within }),
            onTick: (node: AddressNode, ancestors: readonly string[], ranged: boolean, scope: AddressScope) => {
                const key = keyOf(node);
                const row = rowOf(node, ancestors);
                const shownByKey = new Map(scope.visible.map((entry) => [entry.key, entry] as const));
                const childrenOf = addressChildren(
                    scope.branches,
                    (held) => shownByKey.get(held)?.node.path,
                    (held) => shownByKey.get(held)?.ancestors,
                    typeIdOf,
                    within,
                );
                if (ranged) {
                    const shown = scope.visible.filter((entry) => within === undefined || addressRow(entry.node, entry.ancestors, typeIdOf).typeId === within);
                    const rows = new Map(shown.map((entry) => [entry.key, rowOf(entry.node, entry.ancestors)] as const));
                    onSelection(range(selection, shown.map((entry) => entry.key), key, (held) => rows.get(held), childrenOf));
                    return;
                }
                const pick = within === undefined ? { pageId: key, size: node.pages } : { pageId: key, typeId: within, size: null };
                const scopeOf = within === undefined ? { leaf: node.children === 0 } : { leaf: node.children === 0, typeId: within };
                if (stateOf(selection, key, row, scopeOf) === "on") {
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
        (entries: readonly AddressEntry[]) => {
            onRows(new Map(entries.map((entry) => [entry.key, { path: entry.node.path, row: rowOf(entry.node, entry.ancestors) }] as const)));
        },
        [onRows, rowOf],
    );

    return (
        <AddressTree
            key={`${siteId}:${within ?? ""}`}
            siteId={siteId}
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
