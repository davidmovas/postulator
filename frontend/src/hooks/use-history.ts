import { useState, useCallback, useRef } from "react";

interface HistoryState<T> {
    past: T[];
    present: T;
    future: T[];
}

interface UseHistoryOptions {
    maxHistory?: number;
}

export function useHistory<T>(
    initialState: T,
    options: UseHistoryOptions = {}
) {
    const { maxHistory = 50 } = options;

    const [history, setHistory] = useState<HistoryState<T>>({
        past: [],
        present: initialState,
        future: [],
    });

    // Track if we should record history (skip during undo/redo)
    const skipRecordRef = useRef(false);

    const canUndo = history.past.length > 0;
    const canRedo = history.future.length > 0;

    // Set new state and record in history
    const set = useCallback((newState: T | ((prev: T) => T)) => {
        setHistory((prev) => {
            const nextState = typeof newState === "function"
                ? (newState as (prev: T) => T)(prev.present)
                : newState;

            // If skipping record (during undo/redo), just update present
            if (skipRecordRef.current) {
                skipRecordRef.current = false;
                return { ...prev, present: nextState };
            }

            // Add current state to past, clear future
            const newPast = [...prev.past, prev.present];

            // Limit history size
            if (newPast.length > maxHistory) {
                newPast.shift();
            }

            return {
                past: newPast,
                present: nextState,
                future: [],
            };
        });
    }, [maxHistory]);

    // Undo - go back one step
    const undo = useCallback(() => {
        setHistory((prev) => {
            if (prev.past.length === 0) return prev;

            const newPast = [...prev.past];
            const previousState = newPast.pop()!;

            return {
                past: newPast,
                present: previousState,
                future: [prev.present, ...prev.future],
            };
        });
    }, []);

    // Redo - go forward one step
    const redo = useCallback(() => {
        setHistory((prev) => {
            if (prev.future.length === 0) return prev;

            const newFuture = [...prev.future];
            const nextState = newFuture.shift()!;

            return {
                past: [...prev.past, prev.present],
                present: nextState,
                future: newFuture,
            };
        });
    }, []);

    // Clear history
    const clear = useCallback(() => {
        setHistory((prev) => ({
            past: [],
            present: prev.present,
            future: [],
        }));
    }, []);

    // Reset to new initial state
    const reset = useCallback((newInitialState: T) => {
        setHistory({
            past: [],
            present: newInitialState,
            future: [],
        });
    }, []);

    return {
        state: history.present,
        set,
        undo,
        redo,
        clear,
        reset,
        canUndo,
        canRedo,
        historyLength: history.past.length,
        futureLength: history.future.length,
    };
}
