"use client";

import { SettingsSection } from "./settings-section";
import { RiPaletteLine } from "@remixicon/react";

export function AppearanceSettings() {
    return (
        <SettingsSection
            title="Appearance"
            icon={<RiPaletteLine className="h-5 w-5" />}
        >
            <div className="text-center py-8 text-muted-foreground">
                <RiPaletteLine className="h-12 w-12 mx-auto mb-4 opacity-50" />
                <p>Appearance settings will be available soon</p>
            </div>
        </SettingsSection>
    );
}