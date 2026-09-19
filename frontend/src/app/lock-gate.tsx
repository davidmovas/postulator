import { RouterProvider } from "react-router";

import { copy } from "../copy/index.js";
import { useLockGate } from "../data/lock.js";
import { router } from "./router.js";
import { UnlockScreen } from "./unlock.js";

export function LockGate() {
    const gate = useLockGate();

    if (!gate.ready) {
        return (
            <div className="flex h-full items-center justify-center bg-base-950 text-xs text-ink-400">
                {copy.app.loading}
            </div>
        );
    }

    if (gate.locked) {
        return <UnlockScreen />;
    }

    return <RouterProvider router={router} />;
}
