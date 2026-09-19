import { useVirtualizer } from "@tanstack/react-virtual";
import type { ReactElement, ReactNode, UIEvent } from "react";
import { Fragment, useLayoutEffect, useRef } from "react";

import { cx } from "../../ui/index.js";

const offsets = new Map<string, number>();

export interface VirtualRowsProps {
    count: number;
    rowHeight: number;
    row: (index: number) => ReactNode;
    scrollKey?: string;
    footer?: ReactNode;
    className?: string;
}

export function VirtualRows({
    count,
    rowHeight,
    row,
    scrollKey,
    footer,
    className,
}: VirtualRowsProps): ReactElement {
    const viewport = useRef<HTMLDivElement>(null);
    const restored = useRef(false);
    const virtualizer = useVirtualizer({
        count,
        getScrollElement: () => viewport.current,
        estimateSize: () => rowHeight,
        overscan: 16,
    });

    useLayoutEffect(() => {
        if (restored.current || scrollKey === undefined || count === 0) {
            return;
        }
        restored.current = true;
        const held = offsets.get(scrollKey);
        if (held !== undefined && viewport.current !== null) {
            viewport.current.scrollTop = held;
        }
    }, [count, scrollKey]);

    const remember = (event: UIEvent<HTMLDivElement>): void => {
        if (scrollKey !== undefined) {
            offsets.set(scrollKey, event.currentTarget.scrollTop);
        }
    };

    const items = virtualizer.getVirtualItems();
    const start = items.length === 0 ? 0 : items[0].start;

    return (
        <div
            ref={viewport}
            onScroll={remember}
            className={cx("min-h-0 flex-1 overflow-auto", className)}
        >
            <div role="presentation" style={{ position: "relative", height: `${virtualizer.getTotalSize()}px` }}>
                <div
                    role="presentation"
                    style={{
                        position: "absolute",
                        top: 0,
                        left: 0,
                        width: "100%",
                        transform: `translateY(${start}px)`,
                    }}
                >
                    {items.map((item) => (
                        <Fragment key={String(item.key)}>{row(item.index)}</Fragment>
                    ))}
                </div>
            </div>
            {footer}
        </div>
    );
}
