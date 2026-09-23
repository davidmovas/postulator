import type { ReactElement } from "react";
import { Outlet } from "react-router";

import { copy } from "../../copy/index.js";
import { Screen, Tabs } from "../../ui/index.js";
import { settingsTabs } from "./tabs.js";

export function SettingsScreen(): ReactElement {
    return (
        <Screen title={copy.settings.title} tabs={<Tabs label={copy.nav.settings} items={settingsTabs} />}>
            <Outlet />
        </Screen>
    );
}
