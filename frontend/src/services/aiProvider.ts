import {
  CreateAIProvider,
  DeleteAIProvider,
  GetAIModels,
  GetAIProvider,
  GetAvailableModels,
  ListAIProviders,
  ListActiveAIProviders,
  SetAIProviderStatus,
  UpdateAIProvider,
  ValidateModel,
} from "@/wailsjs/wailsjs/go/app/App";
import { dto } from "@/wailsjs/wailsjs/go/models";
import { unwrapMany, unwrapOne, unwrapString } from "./utils";

export interface AIProvider {
  id: number;
  name: string;
  provider: string;
  model: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface ModelsByProvider {
  openai: string[];
  anthropic: string[];
  google: string[];
}

function mapAIProvider(x: dto.AIProvider): AIProvider {
  return {
      id: x.id,
      name: x.name,
      provider: x.provider,
      model: x.model,
      isActive: x.isActive,
      createdAt: x.createdAt,
      updatedAt: x.updatedAt,
  };
}

function mapModelsByProvider(x: dto.ModelsByProvider): ModelsByProvider {
  return { openai: x.openai, anthropic: x.anthropic, google: x.google };
}

export async function listAIProviders(): Promise<AIProvider[]> {
  const res = await ListAIProviders();
  return unwrapMany<dto.AIProvider>(res).map(mapAIProvider);
}

export async function listActiveAIProviders(): Promise<AIProvider[]> {
  const res = await ListActiveAIProviders();
  return unwrapMany<dto.AIProvider>(res).map(mapAIProvider);
}

export async function getAIProvider(id: number): Promise<AIProvider> {
  const res = await GetAIProvider(id);
  return mapAIProvider(unwrapOne<dto.AIProvider>(res));
}

export async function createAIProvider(input: { name: string; apiKey: string; provider: string, model: string; isActive: boolean }): Promise<string> {
  const payload = new dto.AIProviderCreate({
      name: input.name,
      apiKey: input.apiKey,
      provider: input.provider,
      model: input.model,
      isActive: input.isActive,
  });
  const res = await CreateAIProvider(payload);
  return unwrapString(res);
}

export async function updateAIProvider(input: { id: number; name: string; apiKey?: string;  provider: string, model: string; isActive: boolean }): Promise<string> {
  const payload = new dto.AIProviderUpdate({
      id: input.id,
      name: input.name,
      apiKey: input.apiKey,
      provider: input.provider,
      model: input.model,
      isActive: input.isActive,
  });
  const res = await UpdateAIProvider(payload);
  return unwrapString(res);
}

export async function deleteAIProvider(id: number): Promise<string> {
  const res = await DeleteAIProvider(id);
  return unwrapString(res);
}

export async function getAIModels(): Promise<ModelsByProvider> {
  const res = await GetAIModels();
  return mapModelsByProvider(unwrapOne<dto.ModelsByProvider>(res as any));
}

export async function getAvailableModels(providerName: string): Promise<string[]> {
  const res = await GetAvailableModels(providerName);
  return unwrapMany<string>(res as any);
}

export async function validateModel(providerName: string, model: string): Promise<string> {
  const res = await ValidateModel(providerName, model);
  return unwrapString(res);
}

export async function setAIProviderStatus(id: number, isActive: boolean): Promise<string> {
  const res = await SetAIProviderStatus(id, isActive);
  return unwrapString(res);
}
