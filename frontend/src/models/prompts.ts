import { dto } from "@/wailsjs/wailsjs/go/models";

export interface Prompt {
    id: number;
    name: string;
    systemPrompt: string;
    userPrompt: string;
    placeholders: string[];
    createdAt: string;
    updatedAt: string;
}

export interface PromptCreateInput {
    name: string;
    systemPrompt: string;
    userPrompt: string;
    placeholders: string[];
}

export function mapPrompt(x: dto.Prompt): Prompt {
    return {
        id: x.id,
        name: x.name,
        systemPrompt: x.systemPrompt,
        userPrompt: x.userPrompt,
        placeholders: x.placeholders || [],
        createdAt: x.createdAt,
        updatedAt: x.updatedAt,
    };
}