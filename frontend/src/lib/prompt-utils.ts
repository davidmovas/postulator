export function extractPlaceholders(text: string): string[] {
    const placeholderRegex = /\{\{([^}]+)\}\}/g;
    const matches = text.matchAll(placeholderRegex);
    const placeholders = new Set<string>();

    for (const match of matches) {
        if (match[1]) {
            placeholders.add(match[1].trim());
        }
    }

    return Array.from(placeholders);
}

export function extractPlaceholdersFromPrompts(systemPrompt: string, userPrompt: string): string[] {
    const systemPlaceholders = extractPlaceholders(systemPrompt);
    const userPlaceholders = extractPlaceholders(userPrompt);

    const allPlaceholders = [...systemPlaceholders, ...userPlaceholders];
    return Array.from(new Set(allPlaceholders));
}