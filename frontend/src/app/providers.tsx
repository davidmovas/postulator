import { QueryClientProvider } from "@tanstack/react-query";
import { useState } from "react";

import { EventBridge } from "../data/bridge.js";
import { createQueryClient } from "../data/client.js";
import { LockGate } from "./lock-gate.js";

export function Providers() {
    const [client] = useState(createQueryClient);

    return (
        <QueryClientProvider client={client}>
            <EventBridge />
            <LockGate />
        </QueryClientProvider>
    );
}
