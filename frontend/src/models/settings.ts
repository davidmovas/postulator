import { dto } from "@/wailsjs/wailsjs/go/models";

export interface HealthCheckSettings {
    enabled: boolean;
    intervalMinutes: number;
    minIntervalMinutes: number;
    notifyWhenHidden: boolean;
    notifyAlways: boolean;
    notifyWithSound: boolean;
    notifyOnRecover: boolean;
}

export interface HealthCheckSettingsUpdateInput {
    enabled?: boolean;
    intervalMinutes?: number;
    notifyWhenHidden?: boolean;
    notifyAlways?: boolean;
    notifyWithSound?: boolean;
    notifyOnRecover?: boolean;
}

export function mapHealthCheckSettings(x: dto.HealthCheckSettings): HealthCheckSettings {
    return {
        enabled: x.enabled,
        intervalMinutes: x.interval_minutes,
        minIntervalMinutes: x.min_interval_minutes,
        notifyWhenHidden: x.notify_when_hidden,
        notifyAlways: x.notify_always,
        notifyWithSound: x.notify_with_sound,
        notifyOnRecover: x.notify_on_recover,
    };
}

export function toDtoHealthCheckSettings(input: HealthCheckSettingsUpdateInput): dto.HealthCheckSettings {
    return new dto.HealthCheckSettings({
        enabled: input.enabled,
        interval_minutes: input.intervalMinutes,
        notify_when_hidden: input.notifyWhenHidden,
        notify_always: input.notifyAlways,
        notify_with_sound: input.notifyWithSound,
        notify_on_recover: input.notifyOnRecover,
    });
}

// Dashboard Settings
export interface DashboardSettings {
    autoRefreshEnabled: boolean;
    autoRefreshInterval: number;
    minRefreshInterval: number;
}

export interface DashboardSettingsUpdateInput {
    autoRefreshEnabled?: boolean;
    autoRefreshInterval?: number;
}

export function mapDashboardSettings(x: dto.DashboardSettings): DashboardSettings {
    return {
        autoRefreshEnabled: x.autoRefreshEnabled,
        autoRefreshInterval: x.autoRefreshInterval,
        minRefreshInterval: x.minRefreshInterval,
    };
}

export function toDtoDashboardSettings(input: DashboardSettingsUpdateInput, current: DashboardSettings): dto.DashboardSettings {
    return new dto.DashboardSettings({
        autoRefreshEnabled: input.autoRefreshEnabled ?? current.autoRefreshEnabled,
        autoRefreshInterval: input.autoRefreshInterval ?? current.autoRefreshInterval,
        minRefreshInterval: current.minRefreshInterval,
    });
}