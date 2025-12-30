import { dto } from "@/wailsjs/wailsjs/go/models";

export type PromptCategory = "post_gen" | "page_gen" | "link_suggest" | "link_apply" | "sitemap_gen";

export const PROMPT_CATEGORIES: Record<PromptCategory, string> = {
    post_gen: "Post Generation",
    page_gen: "Page Generation",
    link_suggest: "Link Suggestions",
    link_apply: "Link Insertion",
    sitemap_gen: "Sitemap Structure",
};

// Context field types for v2 prompts
export type ContextFieldType = "checkbox" | "select" | "input" | "textarea";

export interface SelectOption {
    value: string;
    label: string;
}

export interface ContextFieldValue {
    enabled: boolean;
    value?: string;
}

export type ContextConfig = Record<string, ContextFieldValue>;

export interface ContextFieldDefinition {
    key: string;
    label: string;
    description: string;
    type: ContextFieldType;
    options?: SelectOption[];
    defaultValue: string;
    required: boolean;
    categories: string[];
    group?: string;
}

export interface ContextFieldsResponse {
    fields: ContextFieldDefinition[];
    defaultConfig: ContextConfig;
}

// Prompt interface with v2 support
export interface Prompt {
    id: number;
    name: string;
    category: PromptCategory;
    isBuiltin: boolean;
    version: number;
    // V2 fields
    instructions?: string;
    contextConfig?: ContextConfig;
    // Legacy v1 fields
    systemPrompt?: string;
    userPrompt?: string;
    placeholders?: string[];
    createdAt: string;
    updatedAt: string;
}

export interface PromptCreateInput {
    name: string;
    category: PromptCategory;
    version: number;
    // V2 fields
    instructions?: string;
    contextConfig?: ContextConfig;
    // Legacy v1 fields
    systemPrompt?: string;
    userPrompt?: string;
    placeholders?: string[];
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
        version: x.version || 1,
        instructions: x.instructions || "",
        contextConfig: x.contextConfig ? mapContextConfig(x.contextConfig) : undefined,
        systemPrompt: x.systemPrompt || "",
        userPrompt: x.userPrompt || "",
        placeholders: x.placeholders || [],
        createdAt: x.createdAt,
        updatedAt: x.updatedAt,
    };
}

export function mapContextConfig(config: Record<string, dto.ContextFieldValue>): ContextConfig {
    const result: ContextConfig = {};
    for (const [key, value] of Object.entries(config)) {
        result[key] = {
            enabled: value.enabled,
            value: value.value,
        };
    }
    return result;
}

export function mapContextFieldDefinition(def: dto.ContextFieldDefinition): ContextFieldDefinition {
    return {
        key: def.key,
        label: def.label,
        description: def.description,
        type: def.type as ContextFieldType,
        options: def.options?.map(opt => ({
            value: opt.value,
            label: opt.label,
        })),
        defaultValue: def.defaultValue,
        required: def.required,
        categories: def.categories,
        group: def.group,
    };
}

export function mapContextFieldsResponse(resp: dto.ContextFieldsResponse): ContextFieldsResponse {
    return {
        fields: resp.fields?.map(mapContextFieldDefinition) || [],
        defaultConfig: resp.defaultConfig ? mapContextConfig(resp.defaultConfig) : {},
    };
}

// Helper to check if a prompt is v2
export function isV2Prompt(prompt: Prompt): boolean {
    return prompt.version >= 2;
}
