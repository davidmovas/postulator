import { dto } from "@/wailsjs/wailsjs/go/models";
import { ImportAndAssignToSite, ImportTopics } from "@/wailsjs/wailsjs/go/handlers/ImporterHandler";
import { unwrapResponse } from "@/lib/api-utils";
import { ImportResult, mapImportResult } from "@/models/importer";

export const importerService = {
    async importTopics(filePath: string): Promise<ImportResult> {
        const response = await ImportTopics(filePath);
        const result = unwrapResponse<dto.ImportResult>(response);
        return mapImportResult(result);
    },

    async importAndAssignToSite(filePath: string, siteId: number): Promise<ImportResult> {
        const response = await ImportAndAssignToSite(filePath, siteId);
        const result = unwrapResponse<dto.ImportResult>(response);
        return mapImportResult(result);
    }
};