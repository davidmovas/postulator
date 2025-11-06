import { dto } from "@/wailsjs/wailsjs/go/models";
import {
    CreateProvider,
    DeleteProvider,
    GetAvailableModels,
    GetProvider,
    ListActiveProviders,
    ListProviders,
    SetProviderStatus,
    UpdateProvider,
    ValidateModel
} from "@/wailsjs/wailsjs/go/handlers/ProvidersHandler";
import {
    mapProvider,
    mapModel,
    Provider,
    ProviderCreateInput,
    ProviderUpdateInput,
    Model
} from "@/models/providers";
import { unwrapArrayResponse, unwrapResponse } from "@/lib/api-utils";

export const providerService = {
    async createProvider(input: ProviderCreateInput): Promise<string> {
        const payload = new dto.Provider({
            name: input.name,
            type: input.type,
            apiKey: input.apiKey,
            model: input.model,
            isActive: input.isActive,
        });

        const response = await CreateProvider(payload);
        return unwrapResponse<string>(response);
    },

    async getProvider(id: number): Promise<Provider> {
        const response = await GetProvider(id);
        const provider = unwrapResponse<dto.Provider>(response);
        return mapProvider(provider);
    },

    async listProviders(): Promise<Provider[]> {
        const response = await ListProviders();
        const providers = unwrapArrayResponse<dto.Provider>(response);
        return providers.map(mapProvider);
    },

    async listActiveProviders(): Promise<Provider[]> {
        const response = await ListActiveProviders();
        const providers = unwrapArrayResponse<dto.Provider>(response);
        return providers.map(mapProvider);
    },

    async updateProvider(input: ProviderUpdateInput): Promise<string> {
        const payload = new dto.Provider({
            id: input.id,
            name: input.name,
            type: input.type,
            apiKey: input.apiKey,
            model: input.model,
            isActive: input.isActive,
        });

        const response = await UpdateProvider(payload);
        return unwrapResponse<string>(response);
    },

    async deleteProvider(id: number): Promise<string> {
        const response = await DeleteProvider(id);
        return unwrapResponse<string>(response);
    },

    async setProviderStatus(id: number, isActive: boolean): Promise<string> {
        const response = await SetProviderStatus(id, isActive);
        return unwrapResponse<string>(response);
    },

    async getAvailableModels(providerType: string): Promise<Model[]> {
        const response = await GetAvailableModels(providerType);
        const models = unwrapResponse<dto.Model[]>(response);
        return models.map(mapModel);
    },

    async validateModel(providerType: string, model: string): Promise<string> {
        const response = await ValidateModel(providerType, model);
        return unwrapResponse<string>(response);
    },
};