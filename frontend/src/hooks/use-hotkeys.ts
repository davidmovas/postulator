"use client";

import { useEffect, useCallback, useRef } from "react";

export interface HotkeyConfig {
    key: string;
    ctrl?: boolean;
    shift?: boolean;
    alt?: boolean;
    description: string;
    category?: string;
    action: () => void;
}

interface UseHotkeysOptions {
    enabled?: boolean;
    preventDefault?: boolean;
}

export function useHotkeys(
    hotkeys: HotkeyConfig[],
    options: UseHotkeysOptions = {}
) {
    const { enabled = true, preventDefault = true } = options;
    const hotkeysRef = useRef(hotkeys);
    hotkeysRef.current = hotkeys;

    const handleKeyDown = useCallback(
        (e: KeyboardEvent) => {
            // Skip if typing in input/textarea
            if (
                e.target instanceof HTMLInputElement ||
                e.target instanceof HTMLTextAreaElement ||
                (e.target instanceof HTMLElement && e.target.isContentEditable)
            ) {
                return;
            }

            for (const hotkey of hotkeysRef.current) {
                const keyMatch = e.key.toLowerCase() === hotkey.key.toLowerCase();
                const ctrlMatch = hotkey.ctrl ? (e.ctrlKey || e.metaKey) : !(e.ctrlKey || e.metaKey);
                const shiftMatch = hotkey.shift ? e.shiftKey : !e.shiftKey;
                const altMatch = hotkey.alt ? e.altKey : !e.altKey;

                if (keyMatch && ctrlMatch && shiftMatch && altMatch) {
                    if (preventDefault) {
                        e.preventDefault();
                    }
                    hotkey.action();
                    return;
                }
            }
        },
        [preventDefault]
    );

    useEffect(() => {
        if (!enabled) return;

        window.addEventListener("keydown", handleKeyDown);
        return () => window.removeEventListener("keydown", handleKeyDown);
    }, [enabled, handleKeyDown]);

    return hotkeysRef.current;
}

// Helper to format hotkey for display
export function formatHotkey(config: HotkeyConfig): string {
    const parts: string[] = [];
    if (config.ctrl) parts.push("Ctrl");
    if (config.shift) parts.push("Shift");
    if (config.alt) parts.push("Alt");
    parts.push(config.key.toUpperCase());
    return parts.join(" + ");
}

// Group hotkeys by category
export function groupHotkeysByCategory(hotkeys: HotkeyConfig[]): Map<string, HotkeyConfig[]> {
    const groups = new Map<string, HotkeyConfig[]>();

    for (const hotkey of hotkeys) {
        const category = hotkey.category || "General";
        if (!groups.has(category)) {
            groups.set(category, []);
        }
        groups.get(category)!.push(hotkey);
    }

    return groups;
}
