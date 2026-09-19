import type { ReactElement, ReactNode } from "react";

import { cx } from "./cx.js";
import type { IconComponent } from "./icons/index.js";
import type { Tone } from "./tone.js";
import { toneClasses } from "./tone.js";

export interface StatusBadgeProps {
    tone: Tone;
    children: ReactNode;
    icon?: IconComponent;
    dot?: boolean;
    className?: string;
}

export function StatusBadge({ tone, children, icon: Icon, dot = true, className }: StatusBadgeProps): ReactElement {
    const classes = toneClasses[tone];
    return (
        <span
            className={cx(
                "inline-flex h-5 shrink-0 items-center gap-1 rounded-sm border px-2 text-xs font-semibold whitespace-nowrap",
                classes.soft,
                classes.border,
                classes.ink,
                className,
            )}
        >
            {Icon === undefined ? (
                dot ? <span aria-hidden={true} className="h-1 w-1 shrink-0 rounded-full bg-current" /> : null
            ) : (
                <Icon size={13} className="shrink-0" />
            )}
            {children}
        </span>
    );
}

export interface CountBadgeProps {
    tone: Tone;
    count: number;
    className?: string;
}

export function CountBadge({ tone, count, className }: CountBadgeProps): ReactElement {
    const classes = toneClasses[tone];
    return (
        <span
            className={cx(
                "inline-flex h-4 min-w-4 shrink-0 items-center justify-center rounded-full px-1 font-mono text-2xs font-medium",
                classes.soft,
                classes.ink,
                className,
            )}
        >
            {count}
        </span>
    );
}
