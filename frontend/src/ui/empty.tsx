import type { ReactElement, ReactNode } from "react";

import { cx } from "./cx.js";
import type { IconComponent } from "./icons/index.js";

export interface EmptyStateProps {
    icon: IconComponent;
    title: string;
    body: string;
    actions?: ReactNode;
    className?: string;
}

export function EmptyState({ icon: Icon, title, body, actions, className }: EmptyStateProps): ReactElement {
    return (
        <div
            className={cx(
                "flex flex-col items-center gap-2 rounded-lg border border-dashed border-hairline p-4 text-center",
                className,
            )}
        >
            <Icon size={26} className="text-ink-faint" />
            <p className="text-sm font-semibold text-ink">{title}</p>
            <p className="max-w-60 text-xs text-ink-dim">{body}</p>
            {actions === undefined ? null : <div className="mt-0.5 flex flex-wrap justify-center gap-2">{actions}</div>}
        </div>
    );
}
