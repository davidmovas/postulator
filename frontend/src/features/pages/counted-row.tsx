import type { ReactElement } from "react";

import { cx } from "../../ui/index.js";

export interface CountedRowProps {
    label: string;
    count: number | undefined;
    active: boolean;
    disabled: boolean;
    onSelect: () => void;
}

export function CountedRow({ label, count, active, disabled, onSelect }: CountedRowProps): ReactElement {
    return (
        <button
            type="button"
            aria-pressed={active}
            disabled={disabled}
            onClick={onSelect}
            className={cx(
                "flex h-6 w-full items-center justify-between gap-2 rounded-sm px-1.5 text-xs transition-colors duration-100",
                "disabled:cursor-not-allowed disabled:opacity-50",
                active ? "bg-accent-soft text-ink" : "text-ink-dim enabled:hover:bg-inset enabled:hover:text-ink",
            )}
        >
            <span className="truncate">{label}</span>
            <span className="shrink-0 font-mono text-2xs text-ink-faint">{count ?? ""}</span>
        </button>
    );
}
