import { dto } from "@/wailsjs/wailsjs/go/models";

export interface ImportResult {
    totalRead: number
    totalAdded: number
    totalSkipped: number
    added: string[]
    skipped: string[]
    errors: string[]
}

export function mapImportResult(x: dto.ImportResult): ImportResult {
    return {
        totalRead: x.totalRead,
        totalAdded: x.totalAdded,
        totalSkipped: x.totalSkipped,
        added: x.added || [],
        skipped: x.skipped || [],
        errors: x.errors || [],
    };
}


