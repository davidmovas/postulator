"use client";

import { useState, useEffect } from "react";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Input } from "@/components/ui/input";
import { SettingsSection } from "./settings-section";
import { useDashboardSettings } from "@/hooks/use-dashboard-settings";
import { RiDashboardLine } from "@remixicon/react";

export function DashboardSettingsComponent() {
    const { settings, isLoading, updateSettings } = useDashboardSettings();
    const [formData, setFormData] = useState({
        autoRefreshEnabled: true,
        autoRefreshInterval: 30
    });

    useEffect(() => {
        if (settings) {
            setFormData({
                autoRefreshEnabled: settings.autoRefreshEnabled,
                autoRefreshInterval: settings.autoRefreshInterval
            });
        }
    }, [settings]);

    const handleChange = (updates: Partial<typeof formData>) => {
        const newData = { ...formData, ...updates };
        setFormData(newData);
        updateSettings(newData);
    };

    if (isLoading && !settings) {
        return (
            <SettingsSection
                title="Dashboard"
                icon={<RiDashboardLine className="h-5 w-5" />}
            >
                <div className="space-y-4 animate-pulse">
                    <div className="h-4 bg-muted rounded w-3/4"></div>
                    <div className="h-10 bg-muted rounded"></div>
                </div>
            </SettingsSection>
        );
    }

    return (
        <SettingsSection
            title="Dashboard"
            icon={<RiDashboardLine className="h-5 w-5" />}
        >
            <div className="space-y-6">
                {/* Auto Refresh Toggle */}
                <div className="flex items-center justify-between">
                    <div className="space-y-1">
                        <Label htmlFor="auto-refresh-enabled" className="text-base font-medium">
                            Auto Refresh
                        </Label>
                        <p className="text-sm text-muted-foreground">
                            Automatically refresh dashboard data
                        </p>
                    </div>
                    <Switch
                        id="auto-refresh-enabled"
                        checked={formData.autoRefreshEnabled}
                        onCheckedChange={(checked) => handleChange({ autoRefreshEnabled: checked })}
                        disabled={isLoading}
                    />
                </div>

                {formData.autoRefreshEnabled && (
                    <div className="flex items-center justify-between">
                        <div className="space-y-1">
                            <Label htmlFor="refresh-interval" className="font-medium">
                                Refresh Interval
                            </Label>
                            <p className="text-sm text-muted-foreground">
                                How often to refresh dashboard data
                            </p>
                        </div>
                        <div className="flex items-center gap-3">
                            <Input
                                id="refresh-interval"
                                type="number"
                                min={settings?.minRefreshInterval || 10}
                                value={formData.autoRefreshInterval}
                                onChange={(e) => handleChange({
                                    autoRefreshInterval: parseInt(e.target.value) || 10
                                })}
                                disabled={isLoading}
                                className="w-20 text-right"
                            />
                            <span className="text-sm text-muted-foreground w-8">sec</span>
                        </div>
                    </div>
                )}
            </div>
        </SettingsSection>
    );
}
