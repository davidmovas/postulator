import type { KeyboardEvent, MouseEvent, PointerEvent as ReactPointerEvent, ReactElement, Ref } from "react";
import { useCallback, useEffect, useImperativeHandle, useLayoutEffect, useRef } from "react";

import { frameScheduler } from "./scheduler.js";
import type { Point, Size } from "./viewport.js";

export interface PointerInfo extends Point {
    buttons: number;
    button: number;
    shiftKey: boolean;
    ctrlKey: boolean;
    altKey: boolean;
    metaKey: boolean;
    pointerId: number;
    pointerType: string;
}

export interface WheelInfo extends Point {
    deltaX: number;
    deltaY: number;
    shiftKey: boolean;
    ctrlKey: boolean;
    altKey: boolean;
    metaKey: boolean;
}

export interface CanvasHandle {
    redraw(): void;
    size(): Size;
    element(): HTMLCanvasElement | null;
}

export interface CanvasHostProps {
    label: string;
    draw: (context: CanvasRenderingContext2D, size: Size) => void;
    onPointerDown?: (pointer: PointerInfo) => void;
    onPointerMove?: (pointer: PointerInfo) => void;
    onPointerUp?: (pointer: PointerInfo) => void;
    onPointerLeave?: () => void;
    onDoubleClick?: (pointer: PointerInfo) => void;
    onContextMenu?: (pointer: PointerInfo) => void;
    onWheel?: (wheel: WheelInfo) => void;
    onKeyDown?: (event: KeyboardEvent<HTMLCanvasElement>) => void;
    onResize?: (size: Size) => void;
    cursor?: string;
    className?: string;
    ref?: Ref<CanvasHandle>;
}

function mouseOf(event: MouseEvent<HTMLCanvasElement>, pointerId: number, pointerType: string): PointerInfo {
    const bounds = event.currentTarget.getBoundingClientRect();
    return {
        x: event.clientX - bounds.left,
        y: event.clientY - bounds.top,
        buttons: event.buttons,
        button: event.button,
        shiftKey: event.shiftKey,
        ctrlKey: event.ctrlKey,
        altKey: event.altKey,
        metaKey: event.metaKey,
        pointerId,
        pointerType,
    };
}

function pointerOf(event: ReactPointerEvent<HTMLCanvasElement>): PointerInfo {
    return mouseOf(event, event.pointerId, event.pointerType);
}

export function CanvasHost({
    label,
    draw,
    onPointerDown,
    onPointerMove,
    onPointerUp,
    onPointerLeave,
    onDoubleClick,
    onContextMenu,
    onWheel,
    onKeyDown,
    onResize,
    cursor,
    className,
    ref,
}: CanvasHostProps): ReactElement {
    const canvas = useRef<HTMLCanvasElement>(null);
    const measured = useRef<Size>({ width: 0, height: 0 });
    const latestDraw = useRef(draw);
    latestDraw.current = draw;

    const paint = useCallback((): void => {
        const element = canvas.current;
        if (element === null) {
            return;
        }
        const context = element.getContext("2d");
        if (context === null) {
            return;
        }
        const size = measured.current;
        const scale = window.devicePixelRatio || 1;
        context.setTransform(scale, 0, 0, scale, 0, 0);
        context.clearRect(0, 0, size.width, size.height);
        latestDraw.current(context, size);
    }, []);

    const frames = useRef(frameScheduler(paint));

    useImperativeHandle(
        ref,
        () => ({
            redraw: () => {
                frames.current.request();
            },
            size: () => measured.current,
            element: () => canvas.current,
        }),
        [],
    );

    useLayoutEffect(() => {
        const element = canvas.current;
        if (element === null) {
            return undefined;
        }
        const resize = (): void => {
            const bounds = element.getBoundingClientRect();
            const scale = window.devicePixelRatio || 1;
            const size = { width: Math.round(bounds.width), height: Math.round(bounds.height) };
            measured.current = size;
            element.width = Math.round(size.width * scale);
            element.height = Math.round(size.height * scale);
            onResize?.(size);
            paint();
        };
        resize();
        const observer = new ResizeObserver(() => {
            resize();
        });
        observer.observe(element);
        return () => {
            observer.disconnect();
        };
    }, [onResize, paint]);

    useEffect(() => {
        frames.current.request();
    }, [draw]);

    useEffect(() => {
        const scheduler = frames.current;
        return () => {
            scheduler.cancel();
        };
    }, []);

    useEffect(() => {
        const element = canvas.current;
        if (element === null || onWheel === undefined) {
            return undefined;
        }
        const listener = (event: WheelEvent): void => {
            event.preventDefault();
            const bounds = element.getBoundingClientRect();
            onWheel({
                x: event.clientX - bounds.left,
                y: event.clientY - bounds.top,
                deltaX: event.deltaX,
                deltaY: event.deltaY,
                shiftKey: event.shiftKey,
                ctrlKey: event.ctrlKey,
                altKey: event.altKey,
                metaKey: event.metaKey,
            });
        };
        element.addEventListener("wheel", listener, { passive: false });
        return () => {
            element.removeEventListener("wheel", listener);
        };
    }, [onWheel]);

    return (
        <canvas
            ref={canvas}
            role="application"
            aria-label={label}
            tabIndex={0}
            className={className}
            style={{ display: "block", width: "100%", height: "100%", cursor, touchAction: "none", outline: "none" }}
            onPointerDown={(event) => {
                event.currentTarget.setPointerCapture(event.pointerId);
                onPointerDown?.(pointerOf(event));
            }}
            onPointerMove={(event) => {
                onPointerMove?.(pointerOf(event));
            }}
            onPointerUp={(event) => {
                if (event.currentTarget.hasPointerCapture(event.pointerId)) {
                    event.currentTarget.releasePointerCapture(event.pointerId);
                }
                onPointerUp?.(pointerOf(event));
            }}
            onPointerLeave={() => {
                onPointerLeave?.();
            }}
            onDoubleClick={(event) => {
                onDoubleClick?.(mouseOf(event, -1, "mouse"));
            }}
            onContextMenu={(event) => {
                if (onContextMenu === undefined) {
                    return;
                }
                event.preventDefault();
                onContextMenu(mouseOf(event, -1, "mouse"));
            }}
            onKeyDown={onKeyDown}
        />
    );
}
