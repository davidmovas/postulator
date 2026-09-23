import { useEffect, useMemo, useRef, useState } from "react";

import { usePageTree } from "../../data/hooks/pages.js";
import { useSiteOverview } from "../../data/hooks/reports.js";
import { branchIds, countNodes, flattenTree, pathTo } from "./tree-model.js";
import type { TreeRow } from "./tree-model.js";

export const treeFetchLimit = 1500;
const autoExpandLimit = 200;
const childLimit = 200;

export interface TreeView {
    armed: boolean;
    pending: boolean;
    sizing: boolean;
    total: number | undefined;
    nodes: number;
    rows: readonly TreeRow[];
    arm: () => void;
    expandAll: () => void;
    collapseAll: () => void;
    toggle: (pageId: string) => void;
    lift: (parentId: string) => void;
}

export function useTreeView(siteId: string, active: boolean, selectedId: string | null): TreeView {
    const overview = useSiteOverview(active ? siteId : null);
    const total = overview.data?.pages.total;
    const [forced, setForced] = useState(false);
    const armed = active && (forced || (total !== undefined && total <= treeFetchLimit));
    const tree = usePageTree(armed ? siteId : null);
    const roots = tree.data?.roots ?? null;

    const [expanded, setExpanded] = useState<ReadonlySet<string>>(() => new Set<string>());
    const [lifted, setLifted] = useState<ReadonlySet<string>>(() => new Set<string>());
    const seeded = useRef(false);

    useEffect(() => {
        if (roots === null || seeded.current) {
            return;
        }
        seeded.current = true;
        setExpanded(countNodes(roots) <= autoExpandLimit ? new Set(branchIds(roots)) : new Set<string>());
    }, [roots]);

    useEffect(() => {
        if (roots === null || selectedId === null) {
            return;
        }
        const trail = pathTo(roots, selectedId);
        if (trail.length === 0) {
            return;
        }
        setExpanded((held) => {
            if (trail.every((id) => held.has(id))) {
                return held;
            }
            const next = new Set(held);
            for (const id of trail) {
                next.add(id);
            }
            return next;
        });
    }, [roots, selectedId]);

    const rows = useMemo(() => flattenTree(roots, expanded, lifted, childLimit), [roots, expanded, lifted]);
    const nodes = useMemo(() => countNodes(roots), [roots]);

    return {
        armed,
        pending: armed && tree.isPending,
        sizing: active && !armed && overview.isPending,
        total,
        nodes,
        rows,
        arm: () => {
            setForced(true);
        },
        expandAll: () => {
            setExpanded(new Set(branchIds(roots)));
        },
        collapseAll: () => {
            setExpanded(new Set<string>());
            setLifted(new Set<string>());
        },
        toggle: (pageId: string) => {
            setExpanded((held) => {
                const next = new Set(held);
                if (next.has(pageId)) {
                    next.delete(pageId);
                } else {
                    next.add(pageId);
                }
                return next;
            });
        },
        lift: (parentId: string) => {
            setLifted((held) => new Set(held).add(parentId));
        },
    };
}
