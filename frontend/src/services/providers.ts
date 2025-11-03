import { unwrapArrayResponse, unwrapResponse } from "@/lib/utils/error-handling";
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

export const providerService = {
    async createProvider(input: ProviderCreateInput): Promise<void> {
        const payload = new dto.Provider({
            name: input.name,
            type: input.type,
            apiKey: input.apiKey,
            model: input.model,
            isActive: input.isActive,
        });

        const response = await CreateProvider(payload);
        unwrapResponse<string>(response);
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

    async updateProvider(input: ProviderUpdateInput): Promise<void> {
        const payload = new dto.Provider({
            id: input.id,
            name: input.name,
            type: input.type,
            apiKey: input.apiKey,
            model: input.model,
            isActive: input.isActive,
        });

        const response = await UpdateProvider(payload);
        unwrapResponse<string>(response);
    },

    async deleteProvider(id: number): Promise<void> {
        const response = await DeleteProvider(id);
        unwrapResponse<string>(response);
    },

    async setProviderStatus(id: number, isActive: boolean): Promise<void> {
        const response = await SetProviderStatus(id, isActive);
        unwrapResponse<string>(response);
    },

    async getAvailableModels(providerType: string): Promise<Model[]> {
        const response = await GetAvailableModels(providerType);
        const models = unwrapArrayResponse<dto.Model>(response);
        return models.map(mapModel);
    },

    async validateModel(providerType: string, model: string): Promise<void> {
        const response = await ValidateModel(providerType, model);
        unwrapResponse<string>(response);
    },
};