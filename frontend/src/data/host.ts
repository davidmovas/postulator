import { Dialogs } from "@wailsio/runtime";

export interface HostFilter {
    displayName: string;
    pattern: string;
}

export interface HostPickOptions {
    title?: string;
    message?: string;
    buttonText?: string;
    directory?: string;
    filename?: string;
    filters?: readonly HostFilter[];
}

function toFilters(filters: readonly HostFilter[] | undefined) {
    return filters?.map((held) => ({ DisplayName: held.displayName, Pattern: held.pattern }));
}

function normalise(path: string): string | null {
    return path === "" ? null : path;
}

export async function pickOpenFile(options: HostPickOptions = {}): Promise<string | null> {
    const picked = await Dialogs.OpenFile({
        Title: options.title,
        Message: options.message,
        ButtonText: options.buttonText,
        Directory: options.directory,
        Filters: toFilters(options.filters),
        CanChooseFiles: true,
        CanChooseDirectories: false,
        AllowsMultipleSelection: false,
    });
    return normalise(picked);
}

export async function pickSaveFile(options: HostPickOptions = {}): Promise<string | null> {
    const picked = await Dialogs.SaveFile({
        Title: options.title,
        Message: options.message,
        ButtonText: options.buttonText,
        Directory: options.directory,
        Filename: options.filename,
        Filters: toFilters(options.filters),
        CanCreateDirectories: true,
    });
    return normalise(picked);
}
