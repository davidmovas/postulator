import { dto } from "@/wailsjs/wailsjs/go/models";
import {
    CreatePrompt,
    DeletePrompt,
    GetPrompt,
    ListPrompts,
    ListPromptsByCategory,
    UpdatePrompt
} from "@/wailsjs/wailsjs/go/handlers/PromptsHandler";
import {
    mapPrompt,
    Prompt,
    PromptCategory,
    PromptCreateInput,
    PromptUpdateInput
} from "@/models/prompts";
import { unwrapArrayResponse, unwrapResponse } from "@/lib/api-utils";

export const promptService = {
    async createPrompt(input: PromptCreateInput): Promise<void> {
        const payload = new dto.Prompt({
            name: input.name,
            category: input.category,
            systemPrompt: input.systemPrompt,
            userPrompt: input.userPrompt,
            placeholders: input.placeholders,
        });

        const response = await CreatePrompt(payload);
        unwrapResponse<string>(response);
    },

    async getPrompt(id: number): Promise<Prompt> {
        const response = await GetPrompt(id);
        const prompt = unwrapResponse<dto.Prompt>(response);
        return mapPrompt(prompt);
    },

    async listPrompts(): Promise<Prompt[]> {
        const response = await ListPrompts();
        const prompts = unwrapArrayResponse<dto.Prompt>(response);
        return prompts.map(mapPrompt);
    },

    async listPromptsByCategory(category: PromptCategory): Promise<Prompt[]> {
        const response = await ListPromptsByCategory(category);
        const prompts = unwrapArrayResponse<dto.Prompt>(response);
        return prompts.map(mapPrompt);
    },

    async updatePrompt(input: PromptUpdateInput): Promise<void> {
        const payload = new dto.Prompt({
            id: input.id,
            name: input.name,
            category: input.category,
            systemPrompt: input.systemPrompt,
            userPrompt: input.userPrompt,
            placeholders: input.placeholders,
        });

        const response = await UpdatePrompt(payload);
        unwrapResponse<string>(response);
    },

    async deletePrompt(id: number): Promise<void> {
        const response = await DeletePrompt(id);
        unwrapResponse<string>(response);
    },
};