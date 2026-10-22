import { useEffect } from "react";

import type { Identified } from "./review-nav.js";
import { neighbourOf } from "./review-nav.js";

function typing(target: EventTarget | null): boolean {
    if (!(target instanceof HTMLElement)) {
        return false;
    }
    return target.isContentEditable || ["INPUT", "TEXTAREA", "SELECT"].includes(target.tagName);
}

export function useItemKeys(siblings: readonly Identified[], itemId: string, onMove: (itemId: string) => void): void {
    useEffect(() => {
        const onKeyDown = (event: KeyboardEvent): void => {
            if (event.ctrlKey || event.metaKey || event.altKey || typing(event.target)) {
                return;
            }
            const pressed = event.key.toLowerCase();
            if (pressed !== "j" && pressed !== "k") {
                return;
            }
            const next = neighbourOf(siblings, itemId, pressed === "j" ? 1 : -1);
            if (next !== null) {
                event.preventDefault();
                onMove(next);
            }
        };
        window.addEventListener("keydown", onKeyDown);
        return () => {
            window.removeEventListener("keydown", onKeyDown);
        };
    }, [siblings, itemId, onMove]);
}
