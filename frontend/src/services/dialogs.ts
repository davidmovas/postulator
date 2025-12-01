import { dto } from "@/wailsjs/wailsjs/go/models";
import {
    OpenFileDialog,
    OpenMultipleFilesDialog,
    OpenDirectoryDialog,
    SaveFileDialog,
} from "@/wailsjs/wailsjs/go/handlers/DialogsHandler";
import { unwrapResponse } from "@/lib/api-utils";

export interface FileFilter {
    displayName: string;
    pattern: string;
}

export const dialogsService = {
    /**
     * Open a native file picker dialog
     * @param title - Dialog title
     * @param filters - File filters (e.g., [{ displayName: "Text Files", pattern: "*.txt;*.csv" }])
     * @returns Selected file path or empty string if cancelled
     */
    async openFileDialog(title: string, filters: FileFilter[] = []): Promise<string> {
        const dtoFilters = filters.map(f => new dto.FileFilter({
            displayName: f.displayName,
            pattern: f.pattern,
        }));
        const response = await OpenFileDialog(title, dtoFilters);
        return unwrapResponse<string>(response);
    },

    /**
     * Open a native file picker dialog that allows multiple file selection
     * @param title - Dialog title
     * @param filters - File filters
     * @returns Array of selected file paths or empty array if cancelled
     */
    async openMultipleFilesDialog(title: string, filters: FileFilter[] = []): Promise<string[]> {
        const dtoFilters = filters.map(f => new dto.FileFilter({
            displayName: f.displayName,
            pattern: f.pattern,
        }));
        const response = await OpenMultipleFilesDialog(title, dtoFilters);
        return unwrapResponse<string[]>(response);
    },

    /**
     * Open a native directory picker dialog
     * @param title - Dialog title
     * @returns Selected directory path or empty string if cancelled
     */
    async openDirectoryDialog(title: string): Promise<string> {
        const response = await OpenDirectoryDialog(title);
        return unwrapResponse<string>(response);
    },

    /**
     * Open a native save file dialog
     * @param title - Dialog title
     * @param defaultFilename - Default filename
     * @param filters - File filters
     * @returns Selected file path or empty string if cancelled
     */
    async saveFileDialog(title: string, defaultFilename: string = "", filters: FileFilter[] = []): Promise<string> {
        const dtoFilters = filters.map(f => new dto.FileFilter({
            displayName: f.displayName,
            pattern: f.pattern,
        }));
        const response = await SaveFileDialog(title, defaultFilename, dtoFilters);
        return unwrapResponse<string>(response);
    },
};
