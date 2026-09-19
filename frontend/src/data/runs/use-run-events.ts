import { useCallback, useEffect, useSyncExternalStore } from "react";

import type { RunEventsState } from "./log.js";
import { ensureLog, getSnapshot, scheduleCatchUp, subscribe } from "./log.js";

export function useRunEvents(runId: string): RunEventsState {
    const listen = useCallback((onChange: () => void) => subscribe(runId, onChange), [runId]);
    const read = useCallback(() => getSnapshot(runId), [runId]);
    const state = useSyncExternalStore(listen, read, read);

    useEffect(() => {
        ensureLog(runId);
        scheduleCatchUp(runId);
    }, [runId]);

    return state;
}
