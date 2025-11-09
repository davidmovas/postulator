"use client";

import { HealthCheckSettings } from "@/components/settings/health-check-settings";
import { GeneralSettings } from "@/components/settings/general-settings";
import { AppearanceSettings } from "@/components/settings/appearance-settings";
import { useSettingsForm } from "@/hooks/use-settings-form";

export default function SettingsPage() {
    const { isSaving, saveChanges } = useSettingsForm();

    return (
        <div className="p-6">
            {/* Header */}
            <div className="text-center mb-8">
                <h1 className="text-3xl font-bold tracking-tight mb-2">Settings</h1>
                <p className="text-muted-foreground">
                    Manage your application settings and preferences
                </p>
            </div>

            {/* Settings Sections - Centered */}
            <div className="max-w-3xl mx-auto space-y-6">
                {/* General Settings */}
                {/* <GeneralSettings /> */}

                {/* Health Checking Settings */}
                <HealthCheckSettings
                    onSave={saveChanges}
                    isSaving={isSaving}
                />

                {/* Appearance Settings */}
                {/* <AppearanceSettings /> */}
            </div>
        </div>
    );
}