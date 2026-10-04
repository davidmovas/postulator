import { plainText } from "../../domain/text.js";

export interface ProductAttribute {
    name: string;
    value: string;
}

export interface ProductOutputs {
    shortDescription: string;
    specifications: ProductAttribute[];
}

function isRecord(value: unknown): value is Record<string, unknown> {
    return typeof value === "object" && value !== null && !Array.isArray(value);
}

export function productOutputsOf(raw: unknown): ProductOutputs | null {
    if (!isRecord(raw)) {
        return null;
    }
    const short = typeof raw["shortDescription"] === "string" ? plainText(raw["shortDescription"]) : "";
    const rows = Array.isArray(raw["specifications"]) ? (raw["specifications"] as unknown[]) : [];
    const specifications: ProductAttribute[] = [];
    for (const row of rows) {
        if (!isRecord(row)) {
            continue;
        }
        const name = typeof row["name"] === "string" ? row["name"].trim() : "";
        const value = typeof row["value"] === "string" ? row["value"].trim() : "";
        if (name !== "" && value !== "") {
            specifications.push({ name, value });
        }
    }
    return { shortDescription: short, specifications };
}

export function nameDiffers(h1: string, storeName: string): boolean {
    const planned = h1.trim();
    const name = storeName.trim();
    return planned !== "" && name !== "" && planned.toLowerCase() !== name.toLowerCase();
}
