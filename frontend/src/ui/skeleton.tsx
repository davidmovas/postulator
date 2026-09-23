import type { ReactElement } from "react";

import { cx } from "./cx.js";

export interface SkeletonProps {
    width?: string;
    height?: number;
    className?: string;
}

export function Skeleton({ width = "100%", height = 12, className }: SkeletonProps): ReactElement {
    return (
        <span
            aria-hidden={true}
            style={{ width, height: `${height}px`, backgroundSize: "380px 100%" }}
            className={cx(
                "block animate-shimmer rounded-sm",
                "bg-[linear-gradient(90deg,var(--color-inset)_0%,var(--color-raised)_40%,var(--color-inset)_80%)]",
                className,
            )}
        />
    );
}

export interface SkeletonRowsProps {
    rows?: number;
    height?: number;
    label: string;
    className?: string;
}

export function SkeletonRows({ rows = 6, height = 12, label, className }: SkeletonRowsProps): ReactElement {
    return (
        <div role="status" aria-label={label} className={cx("flex flex-col gap-2", className)}>
            {Array.from({ length: rows }, (_unused, index) => (
                <Skeleton key={index} height={height} width={index % 3 === 2 ? "62%" : "100%"} />
            ))}
        </div>
    );
}
