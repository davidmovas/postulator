import { dto } from "@/wailsjs/wailsjs/go/models";
import {
    GetHealthCheckSettings,
    UpdateHealthCheckSettings,
} from "@/wailsjs/wailsjs/go/handlers/SettingsHandler";
import {
    HealthCheckSettings,
    HealthCheckSettingsUpdateInput,
    mapHealthCheckSettings,
    toDtoHealthCheckSettings
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
};