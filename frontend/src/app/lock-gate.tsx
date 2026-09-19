import { RouterProvider } from "react-router";

import { copy } from "../copy/index.js";
import { useLockGate } from "../data/lock.js";
import { Spinner } from "../ui/index.js";
import { router } from "./router.js";
import { UnlockScreen } from "./unlock.js";

export function LockGate() {
    const gate = useLockGate();

    if (!gate.ready) {
        return (
            <div className="flex h-full items-center justify-center gap-2 bg-canvas text-xs text-ink-dim">
                <Spinner size={12} />
                {copy.app.loading}
            </div>
        );
    }

    if (gate.locked) {
        return <UnlockScreen />;
    }

    return <RouterProvider router={router} />;
}
