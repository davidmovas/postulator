import type { ReactElement, ReactNode } from "react";

import { cx } from "./cx.js";
import type { Tone } from "./tone.js";
import { toneClasses } from "./tone.js";

function ratio(value: number, max: number): number {
    if (!Number.isFinite(value) || !Number.isFinite(max) || max <= 0) {
        return 0;
    }
    return Math.min(Math.max(value / max, 0), 1);
}

export interface ProgressBarProps {
    value: number;
    max: number;
    label: string;
    tone?: Tone;
    leading?: ReactNode;
    trailing?: ReactNode;
    className?: string;
}

export function ProgressBar({
    value,
    max,
    label,
    tone = "accent",
    leading,
    trailing,
    className,
}: ProgressBarProps): ReactElement {
    const filled = ratio(value, max);
    return (
        <div className={cx("flex flex-col gap-1.5", className)}>
            {leading === undefined && trailing === undefined ? null : (
                <div className="flex justify-between gap-2 font-mono text-xs text-ink-dim">
                    <span className="truncate">{leading}</span>
                    <span className="shrink-0">{trailing}</span>
                </div>
            )}
            <div
                role="progressbar"
                aria-label={label}
                aria-valuenow={value}
                aria-valuemin={0}
                aria-valuemax={max}
                className="h-1 overflow-hidden rounded-sm bg-raised"
            >
                <div
                    style={{ width: `${(filled * 100).toFixed(2)}%` }}
                    className={cx("h-full transition-[width] duration-150 ease-swift", toneClasses[tone].solid)}
                />
            </div>
        </div>
    );
}
