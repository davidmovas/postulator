"use client";

import { HealthCheckSettings } from "@/components/settings/health-check-settings";
import { ProxySettings } from "@/components/settings/proxy-settings";
import { DashboardSettingsComponent } from "@/components/settings/dashboard-settings";
import { useSettingsForm } from "@/hooks/use-settings-form";

export default function SettingsPage() {
    const { isSaving, saveChanges } = useSettingsForm();

    return (
        <div className="p-6">
            <div className="text-center mb-8">
                <h1 className="text-3xl font-bold tracking-tight mb-2">Settings</h1>
                <p className="text-muted-foreground">
                    Manage your application settings and preferences
                </p>
            </div>

            <div className="max-w-3xl mx-auto space-y-6">
                <DashboardSettingsComponent />

                <HealthCheckSettings
                    onSave={saveChanges}
                    isSaving={isSaving}
                />

                <ProxySettings />
            </div>
        </div>
    );
}