import type { ReactElement, ReactNode } from "react";
import { useEffect, useMemo, useState } from "react";

import { copy } from "../../../copy/index.js";
import { usePageTree } from "../../../data/hooks/pages.js";
import type { Page } from "../../../data/types.js";
import { Button, Input, SectionLabel, SkeletonRows } from "../../../ui/index.js";
import { branchIds, countNodes, flattenTree } from "../tree-model.js";
import type { Keep } from "./model.js";
import { branchOf, everyPage, indexTree, narrowTree, tick, toggled } from "./model.js";
import { PageRow } from "./row.js";

const openAllUpTo = 200;
const childLimit = 200;

export interface PageTreeProps {
    siteId: string;
    selected: ReadonlySet<string>;
    onChange: (next: ReadonlySet<string>) => void;
    pickable: Keep;
    keep?: Keep;
    required?: ReadonlyMap<string, string>;
    label: string;
    picked: string;
    refusalOf?: (page: Page) => string | null;
    noteOf?: (page: Page) => string | null;
    filter?: ReactNode;
    problem?: string | null;
}

const noRequired: ReadonlyMap<string, string> = new Map();

export function PageTree({
    siteId,
    selected,
    onChange,
    pickable,
    keep = everyPage,
    required = noRequired,
    label,
    picked,
    refusalOf,
    noteOf,
    filter,
    problem,
}: PageTreeProps): ReactElement {
    const tree = usePageTree(siteId);
    const roots = tree.data?.roots ?? null;
    const [query, setQuery] = useState("");
    const [expanded, setExpanded] = useState<ReadonlySet<string>>(new Set());
    const [lifted, setLifted] = useState<ReadonlySet<string>>(new Set());

    useEffect(() => {
        if (roots !== null && countNodes(roots) <= openAllUpTo) {
            setExpanded(new Set(branchIds(roots)));
        }
    }, [roots]);

    const index = useMemo(() => indexTree(roots), [roots]);
    const narrowed = useMemo(() => narrowTree(roots, query, keep), [roots, query, keep]);
    const filtering = query.trim() !== "" || keep !== everyPage;
    const open = useMemo(
        () => (filtering ? new Set(branchIds(narrowed)) : expanded),
        [filtering, narrowed, expanded],
    );
    const rows = useMemo(() => flattenTree(narrowed, open, lifted, childLimit), [narrowed, open, lifted]);

    return (
        <div className="flex flex-col gap-1.5">
            <div className="flex items-center justify-between gap-2">
                <SectionLabel>{label}</SectionLabel>
                <span className="font-mono text-2xs text-ink-faint">{picked}</span>
            </div>
            <div className="flex items-center gap-2">
                <div className="min-w-0 flex-1">
                    <Input
                        type="search"
                        value={query}
                        placeholder={copy.pages.pick.searchHint}
                        aria-label={copy.pages.pick.search}
                        onChange={(event) => {
                            setQuery(event.target.value);
                        }}
                    />
                </div>
                {filter}
            </div>
            <div className="flex max-h-72 min-h-32 flex-col overflow-auto rounded-md border border-hairline bg-inset p-1">
                {tree.isPending ? (
                    <SkeletonRows rows={5} label={copy.pages.loading} />
                ) : rows.length === 0 ? (
                    <p className="p-1 text-xs text-ink-dim">{filtering ? copy.pages.pick.noMatch : copy.empty.pages}</p>
                ) : (
                    rows.map((row) =>
                        row.kind === "more" ? (
                            <div key={row.id} style={{ paddingLeft: row.depth * 14 }}>
                                <Button
                                    size="sm"
                                    variant="ghost"
                                    onClick={() => {
                                        setLifted((held) => new Set([...held, row.parentId]));
                                    }}
                                >
                                    {copy.pages.tree.showMore(row.hidden)}
                                </Button>
                            </div>
                        ) : (
                            <PageRow
                                key={row.id}
                                page={row.page}
                                depth={row.depth}
                                childCount={row.childCount}
                                expanded={row.expanded}
                                tick={tick(row.id, selected, required, index)}
                                refusal={pickable(row.page) ? (refusalOf?.(row.page) ?? null) : (refusalOf?.(row.page) ?? copy.pages.pick.unavailable)}
                                note={noteOf?.(row.page) ?? null}
                                onToggle={() => {
                                    onChange(toggled(selected, [row.id], !selected.has(row.id)));
                                }}
                                onExpand={() => {
                                    setExpanded((held) => toggled(held, [row.id], !held.has(row.id)));
                                }}
                                onBranch={() => {
                                    const branch = branchOf(row.id, index, pickable);
                                    onChange(toggled(selected, branch, !branch.every((id) => selected.has(id))));
                                }}
                            />
                        ),
                    )
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
