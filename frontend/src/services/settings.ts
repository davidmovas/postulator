import { dto } from "@/wailsjs/wailsjs/go/models";
import {
    GetHealthCheckSettings,
    UpdateHealthCheckSettings,
    GetDashboardSettings,
    UpdateDashboardSettings,
} from "@/wailsjs/wailsjs/go/handlers/SettingsHandler";
import {
    HealthCheckSettings,
    HealthCheckSettingsUpdateInput,
    mapHealthCheckSettings,
    toDtoHealthCheckSettings,
    DashboardSettings,
    DashboardSettingsUpdateInput,
    mapDashboardSettings,
    toDtoDashboardSettings,
} from "@/models/settings";
import { unwrapResponse } from "@/lib/api-utils";

export const settingsService = {
    async getHealthCheckSettings(): Promise<HealthCheckSettings> {
        const response = await GetHealthCheckSettings();
        const settings = unwrapResponse<dto.HealthCheckSettings>(response);
        return mapHealthCheckSettings(settings);
    },

    async updateHealthCheckSettings(input: HealthCheckSettingsUpdateInput): Promise<string> {
        const payload = toDtoHealthCheckSettings(input);
        const response = await UpdateHealthCheckSettings(payload);
        return unwrapResponse<string>(response);
    },

    async getDashboardSettings(): Promise<DashboardSettings> {
        const response = await GetDashboardSettings();
        const settings = unwrapResponse<dto.DashboardSettings>(response);
        return mapDashboardSettings(settings);
    },

    async updateDashboardSettings(input: DashboardSettingsUpdateInput, current: DashboardSettings): Promise<string> {
        const payload = toDtoDashboardSettings(input, current);
        const response = await UpdateDashboardSettings(payload);
        return unwrapResponse<string>(response);
    },
};