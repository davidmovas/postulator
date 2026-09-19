type Frame = (callback: (time: number) => void) => number;

if (typeof globalThis.requestAnimationFrame !== "function") {
  const request: Frame = (callback) =>
    setTimeout(() => {
      callback(0);
    }, 0) as unknown as number;

  globalThis.requestAnimationFrame = request as typeof globalThis.requestAnimationFrame;
  globalThis.cancelAnimationFrame = ((handle: number) => {
    clearTimeout(handle as unknown as ReturnType<typeof setTimeout>);
  }) as typeof globalThis.cancelAnimationFrame;
}
