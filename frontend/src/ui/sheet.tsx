import type { KeyboardEvent, PointerEvent as ReactPointerEvent, ReactElement, ReactNode } from "react";
import { useCallback, useRef } from "react";

import { IconButton } from "./button.js";
import { cx } from "./cx.js";
import { CloseIcon } from "./icons/index.js";

const headroom = 80;
const nudge = 24;

export interface SheetProps {
    open: boolean;
    title: string;
    closeLabel: string;
    onClose: () => void;
    height?: number;
    minHeight?: number;
    resizeLabel?: string;
    onHeightChange?: (height: number) => void;
    header?: ReactNode;
    children: ReactNode;
    className?: string;
    onKeyDown?: (event: KeyboardEvent<HTMLElement>) => void;
}

export function Sheet({
    open,
    title,
    closeLabel,
    onClose,
    height = 240,
    minHeight = 160,
    resizeLabel,
    onHeightChange,
    header,
    children,
    className,
    onKeyDown,
}: SheetProps): ReactElement | null {
    const frame = useRef<HTMLElement>(null);

    const clamp = useCallback(
        (next: number): number => {
            const host = frame.current?.parentElement?.clientHeight ?? 0;
            const ceiling = host > 0 ? Math.max(minHeight, host - headroom) : next;
            return Math.min(Math.max(next, minHeight), ceiling);
        },
        [minHeight],
    );

    const drag = (event: ReactPointerEvent<HTMLDivElement>): void => {
        if (onHeightChange === undefined) {
            return;
        }
        event.preventDefault();
        const from = event.clientY;
        const start = frame.current?.clientHeight ?? height;
        const move = (moved: PointerEvent): void => {
            onHeightChange(clamp(start + from - moved.clientY));
        };
        const stop = (): void => {
            window.removeEventListener("pointermove", move);
        };
        window.addEventListener("pointermove", move);
        window.addEventListener("pointerup", stop, { once: true });
    };

    const nudgeBy = (amount: number): void => {
        onHeightChange?.(clamp((frame.current?.clientHeight ?? height) + amount));
    };

    if (!open) {
        return null;
    }

    return (
        <section
            ref={frame}
            aria-label={title}
            tabIndex={-1}
            className={cx("absolute inset-x-0 bottom-0 z-10 flex flex-col border-t border-edge bg-panel outline-none", className)}
            style={{ height, maxHeight: "100%" }}
            onKeyDown={(event) => {
                if (event.key === "Escape") {
                    event.stopPropagation();
                    onClose();
                    return;
                }
                onKeyDown?.(event);
            }}
        >
            {onHeightChange === undefined ? null : (
                <div
                    role="separator"
                    aria-orientation="horizontal"
                    aria-label={resizeLabel}
                    tabIndex={0}
                    className="absolute inset-x-0 -top-1 z-10 h-2 cursor-ns-resize bg-transparent transition-colors duration-100 hover:bg-accent-border focus-visible:bg-accent-border"
                    onPointerDown={drag}
                    onKeyDown={(event) => {
                        if (event.key === "ArrowUp") {
                            event.preventDefault();
                            nudgeBy(nudge);
                        } else if (event.key === "ArrowDown") {
                            event.preventDefault();
                            nudgeBy(-nudge);
                        }
                    }}
                />
            )}
            <header className="flex h-8 shrink-0 items-center gap-3 border-b border-hairline pr-1 pl-3">
                <span className="shrink-0 text-xs font-semibold text-ink">{title}</span>
                <div className="flex min-w-0 flex-1 items-center gap-2">{header}</div>
                <IconButton icon={CloseIcon} label={closeLabel} variant="ghost" size="sm" onClick={onClose} />
            </header>
            <div className="flex min-h-0 flex-1 flex-col">{children}</div>
        </section>
    );
}
