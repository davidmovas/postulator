import type { ReactElement } from "react";

export interface KbdProps {
    keys: readonly string[];
}

export function Kbd({ keys }: KbdProps): ReactElement {
    return (
        <span className="inline-flex shrink-0 items-center gap-0.5">
            {keys.map((key) => (
                <kbd
                    key={key}
                    className="rounded-sm border border-b-2 border-edge bg-raised-strong px-1 py-px font-mono text-2xs font-medium text-ink-dim"
                >
                    {key}
                </kbd>
            ))}
        </span>
    );
}
