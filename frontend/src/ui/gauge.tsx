import type { ReactElement, ReactNode } from "react";

import { cx } from "./cx.js";

export interface BudgetGaugeProps {
    value: number;
    max: number;
    hardStop?: number;
    label: string;
    leading?: ReactNode;
    trailing?: ReactNode;
    footLeading?: ReactNode;
    footTrailing?: ReactNode;
    className?: string;
}

export function BudgetGauge({
    value,
    max,
    hardStop,
    label,
    leading,
    trailing,
    footLeading,
    footTrailing,
    className,
}: BudgetGaugeProps): ReactElement {
    const usable = Number.isFinite(max) && max > 0 ? max : 0;
    const filled = usable === 0 ? 0 : Math.min(Math.max(value / usable, 0), 1);
    const stop = hardStop === undefined || usable === 0 ? null : Math.min(Math.max(hardStop / usable, 0), 1);
    const over = stop !== null && filled >= stop;

    return (
        <div className={cx("flex flex-col gap-1.5", className)}>
            {leading === undefined && trailing === undefined ? null : (
                <div className="flex justify-between gap-2 font-mono text-xs text-ink-dim">
                    <span className="truncate">{leading}</span>
                    <span className={cx("shrink-0", over ? "text-warn" : undefined)}>{trailing}</span>
                </div>
            )}
            <div
                role="meter"
                aria-label={label}
                aria-valuenow={value}
                aria-valuemin={0}
                aria-valuemax={usable}
                className="relative h-2 overflow-hidden rounded-sm bg-raised"
            >
                <div
                    style={{ width: `${(filled * 100).toFixed(2)}%` }}
                    className="h-full bg-[linear-gradient(90deg,var(--color-accent),var(--color-warn))] transition-[width] duration-150 ease-swift"
                />
                {stop === null ? null : (
                    <span
                        aria-hidden={true}
                        style={{ left: `${(stop * 100).toFixed(2)}%` }}
                        className="absolute -top-px -bottom-px w-px bg-danger"
                    />
                )}
            </div>
            {footLeading === undefined && footTrailing === undefined ? null : (
                <div className="flex justify-between gap-2 font-mono text-2xs text-ink-faint">
                    <span className="truncate">{footLeading}</span>
                    <span className="shrink-0">{footTrailing}</span>
                </div>
            )}
        </div>
    );
}
