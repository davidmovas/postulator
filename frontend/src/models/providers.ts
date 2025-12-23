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

export interface ProviderUpdateInput extends Partial<ProviderCreateInput> {
    id: number;
}

export interface Model {
    id: string;
    name: string;
    provider: string;
    contextWindow: number;
    maxOutputTokens: number;
    inputCost: number;
    outputCost: number;
    rpm: number;
    tpm: number;
    usesCompletionTokens: boolean;
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
        contextWindow: x.contextWindow,
        maxOutputTokens: x.maxOutputTokens,
        inputCost: x.inputCost,
        outputCost: x.outputCost,
        rpm: x.rpm,
        tpm: x.tpm,
        usesCompletionTokens: x.usesCompletionTokens,
    };
}
