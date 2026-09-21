import type { ReactElement } from "react";

import { useTabSettings } from "./schema.js";
import { SettingBody } from "./sections.js";

export function AgentSettingsScreen(): ReactElement {
    const settings = useTabSettings("agent");
    return (
        <div className="mx-auto flex w-full max-w-2xl flex-col gap-4">
            <SettingBody settings={settings} />
        </div>
    );
}
