"use client";

import { SettingsSection } from "./settings-section";
import { RiSettings3Line } from "@remixicon/react";

export function GeneralSettings() {
    return (
        <SettingsSection
            title="General"
            description="Basic application settings"
            icon={<RiSettings3Line className="h-5 w-5" />}
        >
            <div className="text-center py-8 text-muted-foreground">
                <RiSettings3Line className="h-12 w-12 mx-auto mb-4 opacity-50" />
                <p>General settings will be available soon</p>
            </div>
        </SettingsSection>
    );
}