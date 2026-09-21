import type { ReactElement } from "react";

import { BackupPanels } from "./backup.js";
import { MasterPasswordPanel } from "./password.js";

export function SecurityScreen(): ReactElement {
    return (
        <div className="flex max-w-2xl flex-col gap-4">
            <MasterPasswordPanel />
            <BackupPanels />
        </div>
    );
}
