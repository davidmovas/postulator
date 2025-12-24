import { dto } from "@/wailsjs/wailsjs/go/models";

export type PromptCategory = "post_gen" | "page_gen" | "link_suggest" | "link_apply" | "sitemap_gen";

export const PROMPT_CATEGORIES: Record<PromptCategory, string> = {
    post_gen: "Post Generation",
    page_gen: "Page Generation",
    link_suggest: "Link Suggestions",
    link_apply: "Link Insertion",
    sitemap_gen: "Sitemap Structure",
};

export interface Prompt {
    id: number;
    name: string;
    category: PromptCategory;
    isBuiltin: boolean;
    systemPrompt: string;
    userPrompt: string;
    placeholders: string[];
    createdAt: string;
    updatedAt: string;
}

export interface PromptCreateInput {
    name: string;
    category: PromptCategory;
    systemPrompt: string;
    userPrompt: string;
    placeholders: string[];
}

export interface PromptUpdateInput extends Partial<PromptCreateInput> {
    id: number;
}

export function mapPrompt(x: dto.Prompt): Prompt {
    return {
        id: x.id,
        name: x.name,
        category: (x.category || "post_gen") as PromptCategory,
        isBuiltin: x.isBuiltin || false,
        systemPrompt: x.systemPrompt,
        userPrompt: x.userPrompt,
        placeholders: x.placeholders || [],
        createdAt: x.createdAt,
        updatedAt: x.updatedAt,
    };
}