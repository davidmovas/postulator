import {
  CreatePrompt,
  DeletePrompt,
  GetPrompt,
  ListPrompts,
  RenderPrompt,
  UpdatePrompt,
} from "@/wailsjs/wailsjs/go/app/App";
import { dto } from "@/wailsjs/wailsjs/go/models";
import { unwrapMany, unwrapOne, unwrapString } from "./utils";

export interface Prompt {
  id: number;
  name: string;
  systemPrompt: string;
  userPrompt: string;
  placeholders: string[];
  createdAt: string;
  updatedAt: string;
}

export interface PromptRenderResult {
  system: string;
  user: string;
}

export function mapPrompt(x: dto.Prompt): Prompt {
  return {
    id: x.id,
    name: x.name,
    systemPrompt: x.systemPrompt,
    userPrompt: x.userPrompt,
    placeholders: x.placeholders,
    createdAt: x.createdAt,
    updatedAt: x.updatedAt,
  };
}

export function mapPromptRenderResult(x: dto.PromptRenderResult): PromptRenderResult {
  return { system: x.system, user: x.user };
}

export async function listPrompts(): Promise<Prompt[]> {
  const res = await ListPrompts();
  return unwrapMany<dto.Prompt>(res).map(mapPrompt);
}

export async function getPrompt(id: number): Promise<Prompt> {
  const res = await GetPrompt(id);
  return mapPrompt(unwrapOne<dto.Prompt>(res));
}

export async function createPrompt(input: Omit<Prompt, "id" | "createdAt" | "updatedAt">): Promise<string> {
  const payload = new dto.Prompt({
    name: input.name,
    systemPrompt: input.systemPrompt,
    userPrompt: input.userPrompt,
    placeholders: input.placeholders,
  });
  const res = await CreatePrompt(payload);
  return unwrapString(res);
}

export async function updatePrompt(input: Omit<Prompt, "createdAt" | "updatedAt">): Promise<string> {
  const payload = new dto.Prompt({
    id: input.id,
    name: input.name,
    systemPrompt: input.systemPrompt,
    userPrompt: input.userPrompt,
    placeholders: input.placeholders,
  });
  const res = await UpdatePrompt(payload);
  return unwrapString(res);
}

export async function deletePrompt(id: number): Promise<string> {
  const res = await DeletePrompt(id);
  return unwrapString(res);
}

export async function renderPrompt(promptId: number, variables: Record<string, string>): Promise<PromptRenderResult> {
  const res = await RenderPrompt(promptId, variables);
  return mapPromptRenderResult(unwrapOne<dto.PromptRenderResult>(res as any));
}
