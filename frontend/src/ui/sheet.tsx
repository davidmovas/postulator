import type { KeyboardEvent, ReactElement, ReactNode } from "react";

import { IconButton } from "./button.js";
import { cx } from "./cx.js";
import { CloseIcon } from "./icons/index.js";

export interface SheetProps {
    open: boolean;
    title: string;
    closeLabel: string;
    onClose: () => void;
    height?: number;
    header?: ReactNode;
    children: ReactNode;
    className?: string;
    onKeyDown?: (event: KeyboardEvent<HTMLElement>) => void;
}

export function Sheet({ open, title, closeLabel, onClose, height = 240, header, children, className, onKeyDown }: SheetProps): ReactElement | null {
    if (!open) {
        return null;
    }
    return (
        <section
            aria-label={title}
            tabIndex={-1}
            className={cx("absolute inset-x-0 bottom-0 z-10 flex flex-col border-t border-edge bg-panel outline-none", className)}
            style={{ height }}
            onKeyDown={(event) => {
                if (event.key === "Escape") {
                    event.stopPropagation();
                    onClose();
                    return;
                }
                onKeyDown?.(event);
            }}
        >
            <header className="flex h-8 shrink-0 items-center gap-3 border-b border-hairline pr-1 pl-3">
                <span className="text-xs font-semibold text-ink">{title}</span>
                <div className="flex min-w-0 flex-1 items-center gap-2">{header}</div>
                <IconButton icon={CloseIcon} label={closeLabel} variant="ghost" size="sm" onClick={onClose} />
            </header>
            <div className="flex min-h-0 flex-1 flex-col">{children}</div>
        </section>
    );
}
