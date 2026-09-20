import type { ReactElement, ReactNode } from "react";

export interface ToolbarProps {
    label: string;
    children: ReactNode;
}

export function Toolbar({ label, children }: ToolbarProps): ReactElement {
    return (
        <div
            role="toolbar"
            aria-label={label}
            className="flex h-9 shrink-0 items-center gap-2 border-b border-hairline px-3"
        >
            {children}
        </div>
    );
}
