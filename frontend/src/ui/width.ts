import { useCallback, useRef, useState } from "react";

export function useWidthOf(): [(node: HTMLDivElement | null) => void, number] {
    const [width, setWidth] = useState(0);
    const observer = useRef<ResizeObserver | null>(null);
    const attach = useCallback((node: HTMLDivElement | null): void => {
        observer.current?.disconnect();
        observer.current = null;
        if (node === null || typeof ResizeObserver === "undefined") {
            return;
        }
        observer.current = new ResizeObserver((entries) => {
            const box = entries[0]?.contentRect;
            if (box !== undefined) {
                setWidth(box.width);
            }
        });
        observer.current.observe(node);
    }, []);
    return [attach, width];
}
