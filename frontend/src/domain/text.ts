const entities: Readonly<Record<string, string>> = {
    "&amp;": "&",
    "&lt;": "<",
    "&gt;": ">",
    "&quot;": '"',
    "&#39;": "'",
    "&nbsp;": " ",
};

export function plainText(html: string): string {
    return html
        .replace(/<[^>]*>/g, " ")
        .replace(/&(amp|lt|gt|quot|#39|nbsp);/g, (entity) => entities[entity] ?? entity)
        .replace(/\s+/g, " ")
        .trim();
}
