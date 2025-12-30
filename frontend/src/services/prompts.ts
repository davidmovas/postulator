import { dto } from "@/wailsjs/wailsjs/go/models";
import {
    CreatePrompt,
    DeletePrompt,
    GetPrompt,
    ListPrompts,
    ListPromptsByCategory,
    UpdatePrompt,
    GetContextFields,
} from "@/wailsjs/wailsjs/go/handlers/PromptsHandler";
import {
    mapPrompt,
    mapContextFieldsResponse,
    Prompt,
    PromptCategory,
    PromptCreateInput,
    PromptUpdateInput,
    ContextFieldsResponse,
    ContextConfig,
} from "@/models/prompts";
import { unwrapArrayResponse, unwrapResponse } from "@/lib/api-utils";

// Convert frontend ContextConfig to DTO format
function contextConfigToDto(config?: ContextConfig): Record<string, dto.ContextFieldValue> | undefined {
    if (!config) return undefined;
    const result: Record<string, dto.ContextFieldValue> = {};
    for (const [key, value] of Object.entries(config)) {
        result[key] = new dto.ContextFieldValue({
            enabled: value.enabled,
            value: value.value,
        });
    }
    return result;
}

export const promptService = {
    async createPrompt(input: PromptCreateInput): Promise<void> {
        const payload = new dto.Prompt({
            name: input.name,
            category: input.category,
            version: input.version || 2,
            // V2 fields
            instructions: input.instructions,
            contextConfig: contextConfigToDto(input.contextConfig),
            // Legacy v1 fields
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
            version: input.version || 2,
            // V2 fields
            instructions: input.instructions,
            contextConfig: contextConfigToDto(input.contextConfig),
            // Legacy v1 fields
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

    async getContextFields(category: PromptCategory): Promise<ContextFieldsResponse> {
        const response = await GetContextFields(category);
        const data = unwrapResponse<dto.ContextFieldsResponse>(response);
        return mapContextFieldsResponse(data);
    },
};
