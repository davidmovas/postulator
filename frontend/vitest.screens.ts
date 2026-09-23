import { cleanup } from "@testing-library/react";
import { afterEach } from "vitest";

const viewportWidth = 1280;
const viewportHeight = 900;

afterEach(() => {
  cleanup();
});

for (const [property, value] of [
  ["clientWidth", viewportWidth],
  ["clientHeight", viewportHeight],
  ["offsetWidth", viewportWidth],
  ["offsetHeight", viewportHeight],
] as const) {
  Object.defineProperty(globalThis.HTMLElement.prototype, property, {
    configurable: true,
    get(): number {
      return value;
    },
  });
}

globalThis.Element.prototype.getBoundingClientRect = function rect(): DOMRect {
  return {
    x: 0,
    y: 0,
    top: 0,
    left: 0,
    right: viewportWidth,
    bottom: viewportHeight,
    width: viewportWidth,
    height: viewportHeight,
    toJSON: () => ({}),
  } as DOMRect;
};

globalThis.Element.prototype.scrollIntoView = function scroll(): void {};

class Observer {
  private readonly announce: ResizeObserverCallback;

  constructor(announce: ResizeObserverCallback) {
    this.announce = announce;
  }

  observe(target: Element): void {
    this.announce(
      [{ target, contentRect: target.getBoundingClientRect() } as ResizeObserverEntry],
      this as unknown as ResizeObserver,
    );
  }

  unobserve(): void {}

  disconnect(): void {}
}

globalThis.ResizeObserver = Observer as unknown as typeof globalThis.ResizeObserver;

globalThis.matchMedia = ((query: string) => ({
  matches: false,
  media: query,
  onchange: null,
  addListener: () => {},
  removeListener: () => {},
  addEventListener: () => {},
  removeEventListener: () => {},
  dispatchEvent: () => false,
})) as unknown as typeof globalThis.matchMedia;
