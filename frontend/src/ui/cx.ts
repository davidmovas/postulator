export type ClassValue = string | false | null | undefined;

export function cx(...parts: ClassValue[]): string {
    return parts.filter((part): part is string => typeof part === "string" && part.length > 0).join(" ");
}
