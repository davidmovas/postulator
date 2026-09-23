import type { ReactElement } from "react";
import { useEffect, useRef, useState } from "react";

import { copy } from "../../../copy/index.js";
import { cx, SkeletonRows } from "../../../ui/index.js";
import type { Viewport } from "./model.js";
import { frameScale, viewports } from "./model.js";

const gutter = 24;

function useWidthOf(): [(node: HTMLDivElement | null) => void, number] {
    const [width, setWidth] = useState(0);
    const observer = useRef<ResizeObserver | null>(null);
    const attach = (node: HTMLDivElement | null): void => {
        observer.current?.disconnect();
        if (node === null) {
            return;
        }
        observer.current = new ResizeObserver((entries) => {
            const box = entries[0]?.contentRect;
            if (box !== undefined) {
                setWidth(box.width);
            }
        });
        observer.current.observe(node);
    };
    return [attach, width];
}

export interface PreviewFrameProps {
    url: string;
    viewport: Viewport;
    title: string;
}

export function PreviewFrame({ url, viewport, title }: PreviewFrameProps): ReactElement {
    const [attach, available] = useWidthOf();
    const [loaded, setLoaded] = useState(false);
    const width = viewports[viewport];
    const scale = frameScale(available - gutter, width);

    useEffect(() => {
        setLoaded(false);
    }, [url]);

    return (
        <div ref={attach} className="relative flex min-h-0 flex-1 justify-center overflow-hidden bg-canvas p-3">
            <div
                className="relative origin-top overflow-hidden rounded-md border border-edge bg-white"
                style={{ width: `${width}px`, height: `calc(100% / ${scale})`, transform: `scale(${scale})` }}
            >
                {loaded ? null : (
                    <div className="absolute inset-0 bg-panel p-4">
                        <SkeletonRows rows={12} label={copy.pages.preview.loading} />
                    </div>
                )}
                <iframe
                    key={url}
                    src={url}
                    title={title}
                    sandbox="allow-scripts allow-same-origin allow-forms"
                    referrerPolicy="no-referrer"
                    onLoad={() => {
                        setLoaded(true);
                    }}
                    className={cx("h-full w-full border-0", loaded ? "opacity-100" : "opacity-0")}
                />
            </div>
        </div>
    );
}
