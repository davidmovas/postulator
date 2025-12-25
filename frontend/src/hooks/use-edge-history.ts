import { useState, useCallback, useRef } from "react";

// Represents the parent relationship of all nodes
export type EdgeState = Map<number, number | null>; // nodeId -> parentId (null = no parent)

interface HistoryEntry {
    edges: EdgeState;
    timestamp: number;
}

interface UseEdgeHistoryOptions {
    maxHistory?: number;
}

export function useEdgeHistory(options: UseEdgeHistoryOptions = {}) {
    const { maxHistory = 50 } = options;

    const [past, setPast] = useState<HistoryEntry[]>([]);
    const [future, setFuture] = useState<HistoryEntry[]>([]);
    const currentStateRef = useRef<EdgeState>(new Map());

    const canUndo = past.length > 0;
    const canRedo = future.length > 0;

    // Initialize or update current state from nodes
    const setCurrentState = useCallback((edges: EdgeState) => {
        currentStateRef.current = new Map(edges);
    }, []);

    // Record a change (call before making the change)
    const recordChange = useCallback(() => {
        const currentState = currentStateRef.current;
        if (currentState.size === 0) return;

        setPast((prev) => {
            const newPast = [
                ...prev,
                { edges: new Map(currentState), timestamp: Date.now() },
            ];
            // Limit history size
            if (newPast.length > maxHistory) {
                newPast.shift();
            }
            return newPast;
        });
        // Clear future when new change is made
        setFuture([]);
    }, [maxHistory]);

    // Undo - returns the previous edge state
    const undo = useCallback((): EdgeState | null => {
        if (past.length === 0) return null;

        const newPast = [...past];
        const previousEntry = newPast.pop()!;

        // Save current state to future
        setFuture((prev) => [
            { edges: new Map(currentStateRef.current), timestamp: Date.now() },
            ...prev,
        ]);

        setPast(newPast);
        currentStateRef.current = new Map(previousEntry.edges);

        return previousEntry.edges;
    }, [past]);

    // Redo - returns the next edge state
    const redo = useCallback((): EdgeState | null => {
        if (future.length === 0) return null;

        const newFuture = [...future];
        const nextEntry = newFuture.shift()!;

        // Save current state to past
        setPast((prev) => [
            ...prev,
            { edges: new Map(currentStateRef.current), timestamp: Date.now() },
        ]);

        setFuture(newFuture);
        currentStateRef.current = new Map(nextEntry.edges);

        return nextEntry.edges;
    }, [future]);

    // Clear history
    const clear = useCallback(() => {
        setPast([]);
        setFuture([]);
    }, []);

    return {
        setCurrentState,
        recordChange,
        undo,
        redo,
        clear,
        canUndo,
        canRedo,
        historyLength: past.length,
        futureLength: future.length,
    };
}
