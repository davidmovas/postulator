import type { ReactElement } from "react";

import { cx } from "./cx.js";

export interface SpinnerProps {
    size?: number;
    className?: string;
}

export function Spinner({ size = 11, className }: SpinnerProps): ReactElement {
    return (
        <span
            aria-hidden={true}
            className={cx(
                "inline-block shrink-0 animate-spin rounded-full border-[1.5px] border-current border-r-transparent",
                className,
            )}
            style={{ width: `${size}px`, height: `${size}px` }}
        />
    );
}
