"use client";

import { useSearchParams } from "next/navigation";

/**
 * Hook to get a numeric ID from query parameters
 * @param name - Parameter name (default: "id")
 * @returns Parsed number or 0 if not found/invalid
 */
export function useQueryId(name: string = "id"): number {
    const searchParams = useSearchParams();
    const value = searchParams.get(name);

    if (!value) return 0;

    const parsed = parseInt(value, 10);
    return isNaN(parsed) ? 0 : parsed;
}

/**
 * Hook to get a string value from query parameters
 * @param name - Parameter name
 * @param defaultValue - Default value if not found (default: "")
 * @returns String value or default
 */
export function useQueryString(name: string, defaultValue: string = ""): string {
    const searchParams = useSearchParams();
    return searchParams.get(name) ?? defaultValue;
}

/**
 * Hook to get multiple numeric IDs from query parameters
 * @param names - Array of parameter names
 * @returns Object with parameter names as keys and parsed numbers as values
 */
export function useQueryIds<T extends string>(names: T[]): Record<T, number> {
    const searchParams = useSearchParams();

    return names.reduce((acc, name) => {
        const value = searchParams.get(name);
        const parsed = value ? parseInt(value, 10) : 0;
        acc[name] = isNaN(parsed) ? 0 : parsed;
        return acc;
    }, {} as Record<T, number>);
}
