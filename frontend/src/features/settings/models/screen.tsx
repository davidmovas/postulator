import type { ReactElement } from "react";
import { useState } from "react";

import { copy } from "../../../copy/index.js";
import { Button, TuneIcon } from "../../../ui/index.js";
import { AdvancedCard } from "../advanced.js";
import { useTabSettings } from "../schema.js";
import { CatalogDrawer } from "./catalog.js";
import { ProviderCards } from "./keys.js";
import { RoleTable } from "./roles.js";
import { SpendTile } from "./spend.js";

export function ModelSettingsScreen(): ReactElement {
    const settings = useTabSettings("models");
    const [catalogOpen, setCatalogOpen] = useState(false);

    return (
        <div className="@container flex flex-col gap-4">
            <ProviderCards />
            <div className="grid grid-cols-1 items-start gap-4 @3xl:grid-cols-[minmax(0,1fr)_16rem]">
                <div className="flex min-w-0 flex-col gap-4">
                    <RoleTable />
                    <AdvancedCard sections={settings.advanced} />
                </div>
                <div className="flex flex-col gap-4">
                    <SpendTile />
                    <Button
                        variant="secondary"
                        icon={TuneIcon}
                        onClick={() => {
                            setCatalogOpen(true);
                        }}
                    >
                        {copy.settings.models.catalog.open}
                    </Button>
                </div>
            </div>
            {catalogOpen ? (
                <CatalogDrawer
                    onClose={() => {
                        setCatalogOpen(false);
                    }}
                />
            ) : null}
        </div>
    );
}
