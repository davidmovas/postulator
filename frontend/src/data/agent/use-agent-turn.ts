import { useCallback, useSyncExternalStore } from "react";

import type { Turn } from "./turn.js";
import { getTurn, subscribeTurn } from "./turn.js";

export function useAgentTurn(conversationId: string): Turn {
    const listen = useCallback((onChange: () => void) => subscribeTurn(conversationId, onChange), [conversationId]);
    const read = useCallback(() => getTurn(conversationId), [conversationId]);
    return useSyncExternalStore(listen, read, read);
}
