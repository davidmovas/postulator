import type { ReactElement, ReactNode } from "react";

import { cx } from "./cx.js";

export type ScreenVariant = "plain" | "split" | "full";

function given(slot: ReactNode): boolean {
    return slot !== undefined && slot !== null && slot !== false;
}

export interface ScreenProps {
    title: string;
    badge?: ReactNode;
    tabs?: ReactNode;
    actions?: ReactNode;
    toolbar?: ReactNode;
    variant?: ScreenVariant;
    left?: ReactNode;
    right?: ReactNode;
    children: ReactNode;
}

export function Screen({
    title,
    badge,
    tabs,
    actions,
    toolbar,
    variant = "plain",
    left,
    right,
    children,
}: ScreenProps): ReactElement {
    return (
        <div className="flex h-full min-h-0 flex-col">
            <header className="flex h-10 shrink-0 items-center gap-2 border-b border-hairline px-4">
                <h1 className="min-w-0 truncate text-lg font-semibold tracking-tight text-ink">{title}</h1>
                {badge}
                {tabs === undefined ? null : (
                    <div className="flex min-w-0 flex-1 justify-center self-stretch overflow-hidden">{tabs}</div>
                )}
                {actions === undefined ? null : (
                    <div className={cx("flex shrink-0 items-center gap-2", tabs === undefined && "ml-auto")}>
                        {actions}
                    </div>
                )}
            </header>
            {toolbar}
            {variant === "plain" ? (
                <div className="min-h-0 flex-1 overflow-auto p-4">{children}</div>
            ) : (
                <div className="flex min-h-0 flex-1">
                    {given(left) ? (
                        <aside className="w-53 shrink-0 overflow-auto border-r border-hairline">{left}</aside>
                    ) : null}
                    <div className="flex min-w-0 flex-1 flex-col overflow-hidden">{children}</div>
                    {given(right) ? (
                        <aside className="w-80 shrink-0 overflow-auto border-l border-hairline">{right}</aside>
                    ) : null}
                </div>
            )}
        </div>
    );
}
