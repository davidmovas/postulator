import type { ReactElement, ReactNode } from "react";

import { cx } from "./cx.js";

export interface PanelProps {
    children: ReactNode;
    className?: string;
    padded?: boolean;
}

export function Panel({ children, className, padded = false }: PanelProps): ReactElement {
    return (
        <section
            className={cx(
                "flex min-w-0 flex-col overflow-hidden rounded-lg border border-hairline bg-panel",
                padded && "p-4",
                className,
            )}
        >
            {children}
        </section>
    );
}

export interface PanelHeaderProps {
    title: string;
    children?: ReactNode;
    className?: string;
}

export function PanelHeader({ title, children, className }: PanelHeaderProps): ReactElement {
    return (
        <header
            className={cx(
                "flex h-8 shrink-0 items-center justify-between gap-3 border-b border-hairline px-3",
                className,
            )}
        >
            <h2 className="truncate text-2xs font-semibold tracking-label text-ink-faint uppercase">{title}</h2>
            {children}
        </header>
    );
}

export interface SectionLabelProps {
    children: ReactNode;
    className?: string;
}

export function SectionLabel({ children, className }: SectionLabelProps): ReactElement {
    return (
        <div className={cx("text-2xs font-semibold tracking-label text-ink-faint uppercase", className)}>
            {children}
        </div>
    );
}
