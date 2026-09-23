export interface FrameScheduler {
    request(): void;
    cancel(): void;
}

type RequestFrame = (callback: () => void) => number;
type CancelFrame = (handle: number) => void;

export function frameScheduler(
    draw: () => void,
    request: RequestFrame = (callback) => globalThis.requestAnimationFrame(() => callback()),
    cancel: CancelFrame = (handle) => globalThis.cancelAnimationFrame(handle),
): FrameScheduler {
    let handle: number | null = null;
    return {
        request(): void {
            if (handle !== null) {
                return;
            }
            handle = request(() => {
                handle = null;
                draw();
            });
        },
        cancel(): void {
            if (handle !== null) {
                cancel(handle);
                handle = null;
            }
        },
    };
}
