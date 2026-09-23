import type { ReactElement } from "react";

import { useTabSettings } from "./schema.js";
import { SettingBody } from "./sections.js";

export function RunSettingsScreen(): ReactElement {
    const settings = useTabSettings("runs");
    return (
        <div className="mx-auto flex w-full max-w-2xl flex-col gap-4">
            <SettingBody settings={settings} />
        </div>
    );
}
