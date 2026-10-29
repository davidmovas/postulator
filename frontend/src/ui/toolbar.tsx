import type { ReactElement, ReactNode } from "react";

import { cx } from "./cx.js";

export interface ToolbarProps {
    label: string;
    children: ReactNode;
    wrap?: boolean;
}

export function Toolbar({ label, children, wrap = false }: ToolbarProps): ReactElement {
    return (
        <div
            role="toolbar"
            aria-label={label}
            className={cx(
                "flex shrink-0 items-center gap-2 border-b border-hairline px-3",
                wrap ? "min-h-9 flex-wrap gap-y-1 py-1" : "h-9",
            )}
        >
            {children}
        </div>
    );
}
