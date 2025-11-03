import { dto } from "@/wailsjs/wailsjs/go/models";

export interface Provider {
    id: number;
    name: string;
    type: string;
    apiKey: string;
    model: string;
    isActive: boolean;
    createdAt: string;
    updatedAt: string;
}

export interface ProviderCreateInput {
    name: string;
    type: string;
    apiKey: string;
    model: string;
    isActive: boolean;
}

export interface Model {
    id: string;
    name: string;
    provider: string;
    maxTokens: number;
    inputCost: number;
    outputCost: number;
}

export function mapProvider(x: dto.Provider): Provider {
    return {
        id: x.id,
        name: x.name,
        type: x.type,
        apiKey: x.apiKey,
        model: x.model,
        isActive: x.isActive,
        createdAt: x.createdAt,
        updatedAt: x.updatedAt,
    };
}

export function mapModel(x: dto.Model): Model {
    return {
        id: x.id,
        name: x.name,
        provider: x.provider,
        maxTokens: x.maxTokens,
        inputCost: x.inputCost,
        outputCost: x.outputCost,
    };
}
