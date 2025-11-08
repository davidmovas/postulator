"use client";

import { useState, useEffect } from "react";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Input } from "@/components/ui/input";
import { SettingsSection } from "./settings-section";
import { useHealthCheckSettings } from "@/hooks/use-health-check-settings";
import { RiPulseLine, RiNotificationLine, RiSoundModuleLine } from "@remixicon/react";

interface HealthCheckSettingsProps {
    onSave: (section: string, data: any) => void;
    isSaving?: boolean;
}

export function HealthCheckSettings({ onSave, isSaving }: HealthCheckSettingsProps) {
    const { settings, isLoading } = useHealthCheckSettings();
    const [formData, setFormData] = useState({
        enabled: false,
        intervalMinutes: 5,
        notifyWhenHidden: false,
        notifyAlways: false,
        notifyWithSound: false,
        notifyOnRecover: false
    });

    useEffect(() => {
        if (settings) {
            const newFormData = {
                enabled: settings.enabled,
                intervalMinutes: settings.intervalMinutes,
                notifyWhenHidden: settings.notifyWhenHidden,
                notifyAlways: settings.notifyAlways,
                notifyWithSound: settings.notifyWithSound,
                notifyOnRecover: settings.notifyOnRecover
            };
            setFormData(newFormData);
        }
    }, [settings]);

    const handleChange = (updates: Partial<typeof formData>) => {
        const newData = { ...formData, ...updates };
        setFormData(newData);
        onSave('healthCheck', newData);
    };

    if (isLoading && !settings) {
        return (
            <SettingsSection
                title="Health Check"
                icon={<RiPulseLine className="h-5 w-5" />}
            >
                <div className="space-y-4 animate-pulse">
                    <div className="h-4 bg-muted rounded w-3/4"></div>
                    <div className="h-10 bg-muted rounded"></div>
                    <div className="h-10 bg-muted rounded"></div>
                </div>
            </SettingsSection>
        );
    }

    return (
        <SettingsSection
            title="Health Check"
            icon={<RiPulseLine className="h-5 w-5" />}
        >
            <div className="space-y-6">
                {/* Main Toggle */}
                <div className="flex items-center justify-between">
                    <div className="space-y-1">
                        <Label htmlFor="health-check-enabled" className="text-base font-medium">
                            Enable Health Check
                        </Label>
                        <p className="text-sm text-muted-foreground">
                            Automatically monitor site health status in the background
                        </p>
                    </div>
                    <Switch
                        id="health-check-enabled"
                        checked={formData.enabled}
                        onCheckedChange={(checked) => handleChange({ enabled: checked })}
                        disabled={isLoading}
                    />
                </div>

                {formData.enabled && (
                    <>
                        {/* Check Interval */}
                        <div className="flex items-center justify-between">
                            <div className="space-y-1">
                                <Label htmlFor="check-interval" className="font-medium">
                                    Check Interval
                                </Label>
                                <p className="text-sm text-muted-foreground">
                                    How often to automatically check site health status
                                </p>
                            </div>
                            <div className="flex items-center gap-3">
                                <Input
                                    id="check-interval"
                                    type="number"
                                    min={settings?.minIntervalMinutes || 1}
                                    value={formData.intervalMinutes}
                                    onChange={(e) => handleChange({
                                        intervalMinutes: parseInt(e.target.value) || 1
                                    })}
                                    disabled={isLoading}
                                    className="w-20 text-right"
                                />
                                <span className="text-sm text-muted-foreground w-8">min</span>
                            </div>
                        </div>

                        {/* Notification Settings */}
                        <div className="space-y-4 border-t pt-4">
                            <div className="flex items-center gap-2">
                                <RiNotificationLine className="h-4 w-4 text-muted-foreground" />
                                <h4 className="font-medium">Notifications</h4>
                            </div>

                            <div className="space-y-4">
                                <div className="flex items-center justify-between">
                                    <div className="space-y-1">
                                        <Label htmlFor="notify-when-hidden" className="font-normal">
                                            Notify when app is hidden
                                        </Label>
                                        <p className="text-sm text-muted-foreground">
                                            Show notifications even when app is in background
                                        </p>
                                    </div>
                                    <Switch
                                        id="notify-when-hidden"
                                        checked={formData.notifyWhenHidden}
                                        onCheckedChange={(checked) => handleChange({ notifyWhenHidden: checked })}
                                        disabled={isLoading}
                                    />
                                </div>

                                <div className="flex items-center justify-between">
                                    <div className="space-y-1">
                                        <Label htmlFor="notify-always" className="font-normal">
                                            Always notify
                                        </Label>
                                        <p className="text-sm text-muted-foreground">
                                            Notify on every check, not just status changes
                                        </p>
                                    </div>
                                    <Switch
                                        id="notify-always"
                                        checked={formData.notifyAlways}
                                        onCheckedChange={(checked) => handleChange({ notifyAlways: checked })}
                                        disabled={isLoading}
                                    />
                                </div>

                                <div className="flex items-center justify-between">
                                    <div className="space-y-1">
                                        <Label htmlFor="notify-on-recover" className="font-normal">
                                            Notify on recovery
                                        </Label>
                                        <p className="text-sm text-muted-foreground">
                                            Notify when site becomes healthy again
                                        </p>
                                    </div>
                                    <Switch
                                        id="notify-on-recover"
                                        checked={formData.notifyOnRecover}
                                        onCheckedChange={(checked) => handleChange({ notifyOnRecover: checked })}
                                        disabled={isLoading}
                                    />
                                </div>

                                <div className="flex items-center justify-between">
                                    <div className="space-y-1">
                                        <div className="flex items-center gap-2">
                                            <Label htmlFor="notify-with-sound" className="font-normal">
                                                Play sound
                                            </Label>
                                        </div>
                                        <p className="text-sm text-muted-foreground">
                                            Play sound with notifications
                                        </p>
                                    </div>
                                    <Switch
                                        id="notify-with-sound"
                                        checked={formData.notifyWithSound}
                                        onCheckedChange={(checked) => handleChange({ notifyWithSound: checked })}
                                        disabled={isLoading}
                                    />
                                </div>
                            </div>
                        </div>
                    </>
                )}
            </div>
        </SettingsSection>
    );
}