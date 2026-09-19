import type { HTMLAttributes, ReactElement, ReactNode } from "react";
import { createContext, useContext } from "react";

import { cx } from "./cx.js";
import { ArrowUpwardIcon, UnfoldMoreIcon } from "./icons/index.js";

const ColumnsContext = createContext<string>("1fr");

export interface DenseTableProps {
    columns: string;
    children: ReactNode;
    label: string;
    className?: string;
}

export function DenseTable({ columns, children, label, className }: DenseTableProps): ReactElement {
    return (
        <ColumnsContext.Provider value={columns}>
            <div role="table" aria-label={label} className={cx("flex min-w-0 flex-col", className)}>
                {children}
            </div>
        </ColumnsContext.Provider>
    );
}

export interface TableHeadProps {
    children: ReactNode;
    className?: string;
}

export function TableHead({ children, className }: TableHeadProps): ReactElement {
    const columns = useContext(ColumnsContext);
    return (
        <div
            role="row"
            style={{ gridTemplateColumns: columns }}
            className={cx(
                "grid h-6 shrink-0 items-center gap-3 border-b border-hairline bg-inset px-3",
                "text-2xs font-semibold tracking-label text-ink-faint uppercase",
                className,
            )}
        >
            {children}
        </div>
    );
}

export type SortDirection = "asc" | "desc";

export interface SortableHeaderProps {
    children: ReactNode;
    active: boolean;
    direction: SortDirection;
    onToggle: () => void;
    align?: "left" | "right";
    className?: string;
}

export function SortableHeader({
    children,
    active,
    direction,
    onToggle,
    align = "left",
    className,
}: SortableHeaderProps): ReactElement {
    return (
        <button
            type="button"
            role="columnheader"
            aria-sort={active ? (direction === "asc" ? "ascending" : "descending") : "none"}
            onClick={onToggle}
            className={cx(
                "inline-flex min-w-0 items-center gap-1 text-2xs font-semibold tracking-label uppercase",
                align === "right" ? "justify-end" : "justify-start",
                active ? "text-ink-soft" : "text-ink-faint hover:text-ink-soft",
                className,
            )}
        >
            <span className="truncate">{children}</span>
            {active ? (
                <ArrowUpwardIcon
                    size={12}
                    className={cx("shrink-0 transition-transform duration-100", direction === "desc" && "rotate-180")}
                />
            ) : (
                <UnfoldMoreIcon size={12} className="shrink-0 opacity-50" />
            )}
        </button>
    );
}

export interface TableRowProps extends Omit<HTMLAttributes<HTMLDivElement>, "role" | "style"> {
    selected?: boolean;
    interactive?: boolean;
}

export function TableRow({
    selected = false,
    interactive = false,
    className,
    children,
    ...rest
}: TableRowProps): ReactElement {
    const columns = useContext(ColumnsContext);
    return (
        <div
            role="row"
            aria-selected={interactive ? selected : undefined}
            style={{ gridTemplateColumns: columns }}
            className={cx(
                "grid h-7 shrink-0 items-center gap-3 border-b border-inset px-3 text-sm transition-colors duration-100",
                interactive && "cursor-pointer",
                selected
                    ? "bg-accent-soft shadow-[inset_2px_0_0_var(--color-accent)]"
                    : interactive && "hover:bg-inset",
                className,
            )}
            {...rest}
        >
            {children}
        </div>
    );
}

export interface TableCellProps {
    children: ReactNode;
    mono?: boolean;
    align?: "left" | "right";
    muted?: boolean;
    className?: string;
}

export function TableCell({
    children,
    mono = false,
    align = "left",
    muted = false,
    className,
}: TableCellProps): ReactElement {
    return (
        <div
            role="cell"
            className={cx(
                "min-w-0 truncate",
                mono && "font-mono text-xs",
                align === "right" && "text-right",
                muted ? "text-ink-dim" : "text-ink",
                className,
            )}
        >
            {children}
        </div>
    );
}
