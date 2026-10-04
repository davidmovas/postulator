import type { ReactElement } from "react";
import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router";

import { copy } from "../../copy/index.js";
import { useCategoryTree } from "../../data/hooks/pages.js";
import { ChevronRightIcon, cx, SectionLabel } from "../../ui/index.js";
import type { CategoryChip, CategoryRow } from "./category-tree.js";
import { ancestorsOf, categoryChip, categoryRows, categoryTree } from "./category-tree.js";
import { CountedRow } from "./counted-row.js";

const indentStep = 10;

const chipClasses: Readonly<Record<CategoryChip["state"], string>> = {
    onSite: "border-hairline bg-inset text-ink-soft",
    onPublish: "border-dashed border-edge text-ink-dim",
};

interface NodeRowProps {
    row: CategoryRow;
    active: boolean;
    disabled: boolean;
    onToggle: (id: string) => void;
    onSelect: (id: string) => void;
}

function NodeRow({ row, active, disabled, onToggle, onSelect }: NodeRowProps): ReactElement {
    const { node } = row;
    const chip = categoryChip(node);
    const said = copy.pages.categories;
    return (
        <li
            data-category-node={node.id}
            className="flex h-6 items-center gap-0.5"
            style={{ paddingLeft: `${String(row.depth * indentStep)}px` }}
        >
            {row.branches ? (
                <button
                    type="button"
                    aria-expanded={row.open}
                    aria-label={row.open ? said.collapse(node.name) : said.expand(node.name)}
                    disabled={disabled}
                    onClick={() => {
                        onToggle(node.id);
                    }}
                    className="flex h-5 w-4 shrink-0 items-center justify-center rounded-sm text-ink-faint enabled:hover:text-ink disabled:opacity-50"
                >
                    <ChevronRightIcon size={12} className={cx("transition-transform duration-100", row.open && "rotate-90")} />
                </button>
            ) : (
                <span aria-hidden={true} className="w-4 shrink-0" />
            )}
            <button
                type="button"
                aria-pressed={active}
                disabled={disabled}
                title={`${node.name}: ${chip.hint}. ${said.pagesUnder(node.pages)}.`}
                onClick={() => {
                    onSelect(node.id);
                }}
                className={cx(
                    "flex h-6 min-w-0 flex-1 items-center gap-1.5 rounded-sm px-1 text-xs transition-colors duration-100",
                    "disabled:cursor-not-allowed disabled:opacity-50",
                    active ? "bg-accent-soft text-ink" : "text-ink-dim enabled:hover:bg-inset enabled:hover:text-ink",
                )}
            >
                <span className="min-w-0 flex-1 truncate text-left">{node.name}</span>
                <span
                    data-category-chip={chip.state}
                    className={cx(
                        "inline-flex h-4 shrink-0 items-center rounded-sm border px-1 font-mono text-2xs",
                        chipClasses[chip.state],
                    )}
                >
                    {chip.label}
                </span>
                <span className="min-w-4 shrink-0 text-right font-mono text-2xs text-ink-faint">{node.pages}</span>
            </button>
        </li>
    );
}

export interface CategoryFilterProps {
    siteId: string;
    selectedId: string;
    total: number | undefined;
    disabled: boolean;
    onSelect: (categoryId: string) => void;
}

export function CategoryFilter({ siteId, selectedId, total, disabled, onSelect }: CategoryFilterProps): ReactElement {
    const listed = useCategoryTree(siteId);
    const nodes = listed.data?.categories ?? null;
    const tree = useMemo(() => categoryTree(nodes ?? []), [nodes]);
    const [open, setOpen] = useState<ReadonlySet<string>>(() => new Set());
    const said = copy.pages.categories;

    useEffect(() => {
        if (selectedId === "") {
            return;
        }
        const above = ancestorsOf(tree, selectedId);
        setOpen((held) => (above.every((id) => held.has(id)) ? held : new Set([...held, ...above])));
    }, [tree, selectedId]);

    const toggle = (id: string): void => {
        setOpen((held) => {
            const next = new Set(held);
            if (next.has(id)) {
                next.delete(id);
            } else {
                next.add(id);
            }
            return next;
        });
    };

    const rows = categoryRows(tree, open);

    return (
        <div data-category-filter={true} className="flex flex-col gap-1">
            <SectionLabel>{said.title}</SectionLabel>
            {nodes === null ? (
                <p className="px-1.5 text-2xs text-ink-faint">{listed.isPending ? said.loading : ""}</p>
            ) : rows.length === 0 ? (
                <p className="flex flex-col gap-1 px-1.5 text-2xs text-ink-faint">
                    <span>{said.empty}</span>
                    <Link to={`/s/${siteId}/import`} className="w-fit">
                        {said.importSheet}
                    </Link>
                </p>
            ) : (
                <>
                    <CountedRow
                        label={said.all}
                        count={total}
                        active={selectedId === ""}
                        disabled={disabled}
                        onSelect={() => {
                            onSelect("");
                        }}
                    />
                    <ul aria-label={said.title} className="flex flex-col">
                        {rows.map((row) => (
                            <NodeRow
                                key={row.node.id}
                                row={row}
                                active={row.node.id === selectedId}
                                disabled={disabled}
                                onToggle={toggle}
                                onSelect={onSelect}
                            />
                        ))}
                    </ul>
                    <p className="px-1.5 text-2xs text-ink-faint">{said.legend}</p>
                </>
            )}
        </div>
    );
}
