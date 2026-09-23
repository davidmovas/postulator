import type { ReactElement } from "react";
import { useEffect, useMemo, useState } from "react";

import { copy } from "../../copy/index.js";
import { usePageTree } from "../../data/hooks/pages.js";
import { Button, Input, Segmented, SectionLabel, SkeletonRows } from "../../ui/index.js";
import type { SegmentedOption } from "../../ui/index.js";
import { branchIds, countNodes, flattenTree } from "../pages/tree-model.js";
import type { Scope } from "./start-selection.js";
import { branchOf, indexTree, narrowTree, pickable, requiredParents, scopes, tick } from "./start-selection.js";
import { TargetRow } from "./start-tree-row.js";

const openAllUpTo = 200;
const childLimit = 200;

const scopeOptions: readonly SegmentedOption<Scope>[] = scopes.map((scope) => ({
    value: scope,
    label: copy.runs.start.scopes[scope],
}));

export interface TargetTreeProps {
    siteId: string;
    selected: ReadonlySet<string>;
    problem?: string | null;
    onChange: (next: ReadonlySet<string>) => void;
}

function toggled(held: ReadonlySet<string>, ids: readonly string[], on: boolean): ReadonlySet<string> {
    const next = new Set(held);
    for (const id of ids) {
        if (on) {
            next.add(id);
        } else {
            next.delete(id);
        }
    }
    return next;
}

export function TargetTree({ siteId, selected, problem, onChange }: TargetTreeProps): ReactElement {
    const tree = usePageTree(siteId);
    const roots = tree.data?.roots ?? null;
    const [query, setQuery] = useState("");
    const [scope, setScope] = useState<Scope>("all");
    const [expanded, setExpanded] = useState<ReadonlySet<string>>(new Set());
    const [lifted, setLifted] = useState<ReadonlySet<string>>(new Set());

    useEffect(() => {
        if (roots !== null && countNodes(roots) <= openAllUpTo) {
            setExpanded(new Set(branchIds(roots)));
        }
    }, [roots]);

    const index = useMemo(() => indexTree(roots), [roots]);
    const required = useMemo(() => requiredParents(selected, index), [selected, index]);
    const narrowed = useMemo(() => narrowTree(roots, query, scope), [roots, query, scope]);
    const filtering = query.trim() !== "" || scope !== "all";
    const open = useMemo(
        () => (filtering ? new Set(branchIds(narrowed)) : expanded),
        [filtering, narrowed, expanded],
    );
    const rows = useMemo(() => flattenTree(narrowed, open, lifted, childLimit), [narrowed, open, lifted]);

    return (
        <div className="flex flex-col gap-1.5">
            <div className="flex items-center justify-between gap-2">
                <SectionLabel>{copy.runs.start.pages}</SectionLabel>
                <span className="font-mono text-2xs text-ink-faint">
                    {copy.runs.start.picked(selected.size, required.size)}
                </span>
            </div>
            <div className="flex items-center gap-2">
                <div className="min-w-0 flex-1">
                    <Input
                        type="search"
                        value={query}
                        placeholder={copy.runs.start.searchHint}
                        aria-label={copy.runs.start.search}
                        onChange={(event) => {
                            setQuery(event.target.value);
                        }}
                    />
                </div>
                <Segmented
                    label={copy.runs.start.scope}
                    size="sm"
                    value={scope}
                    options={scopeOptions}
                    onValueChange={setScope}
                />
            </div>
            <div className="flex max-h-72 min-h-32 flex-col overflow-auto rounded-md border border-hairline bg-inset p-1">
                {tree.isPending ? (
                    <SkeletonRows rows={5} label={copy.pages.loading} />
                ) : rows.length === 0 ? (
                    <p className="p-1 text-xs text-ink-dim">{filtering ? copy.runs.start.noMatch : copy.empty.pages}</p>
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
                            <TargetRow
                                key={row.id}
                                page={row.page}
                                depth={row.depth}
                                childCount={row.childCount}
                                expanded={row.expanded}
                                tick={tick(row.id, selected, required, index)}
                                neededBy={required.get(row.id) ?? null}
                                writable={pickable(row.page)}
                                onToggle={() => {
                                    onChange(toggled(selected, [row.id], !selected.has(row.id)));
                                }}
                                onExpand={() => {
                                    setExpanded((held) => toggled(held, [row.id], !held.has(row.id)));
                                }}
                                onBranch={() => {
                                    const branch = branchOf(row.id, index);
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
