import { useMutation, useQueryClient } from "@tanstack/react-query";

import {
    deleteProviderKey,
    exportBackup,
    getSetting,
    importBackup,
    lock,
    providerKeys,
    setMasterPassword,
    setProviderKey,
    setSetting,
    settingsSchema,
    unlock,
} from "../endpoints/settings.js";
import { keys } from "../keys.js";
import { markLocked, markUnlocked, useLockState } from "../lock.js";
import { useUnlockedQuery } from "../query.js";

export { useLockState };

export function useSettingsSchema() {
    return useUnlockedQuery({
        queryKey: keys.settings.schema(),
        queryFn: ({ signal }) => settingsSchema({}, signal),
    });
}

export function useSetting(key: string | null) {
    return useUnlockedQuery({
        queryKey: keys.settings.value(key ?? ""),
        queryFn: ({ signal }) => getSetting({ key: key ?? "" }, signal),
        enabled: key !== null && key !== "",
    });
}

export function useProviderKeys() {
    return useUnlockedQuery({
        queryKey: keys.models.providerKeys(),
        queryFn: ({ signal }) => providerKeys({}, signal),
    });
}

export function useSetSetting() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof setSetting>[0]) => setSetting(request),
        onSuccess: (answered) => {
            client.setQueryData(keys.settings.value(answered.key), {
                key: answered.key,
                value: answered.value,
                isDefault: false,
            });
        },
    });
}

export function useSetProviderKey() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof setProviderKey>[0]) => setProviderKey(request),
        onSuccess: () => {
            void client.invalidateQueries({ queryKey: keys.models.root() });
        },
    });
}

export function useDeleteProviderKey() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof deleteProviderKey>[0]) => deleteProviderKey(request),
        onSuccess: () => {
            void client.invalidateQueries({ queryKey: keys.models.root() });
        },
    });
}

export function useLock() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: () => lock({}),
        onSuccess: () => {
            markLocked(client);
        },
    });
}

export function useUnlock() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof unlock>[0]) => unlock(request),
        onSuccess: (answered) => {
            markUnlocked(client, answered);
        },
    });
}

export function useSetMasterPassword() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof setMasterPassword>[0]) => setMasterPassword(request),
        onSuccess: (answered) => {
            client.setQueryData(keys.settings.lock(), answered);
        },
    });
}

export function useExportBackup() {
    return useMutation({
        mutationFn: (request: Parameters<typeof exportBackup>[0]) => exportBackup(request),
    });
}

export function useImportBackup() {
    const client = useQueryClient();
    return useMutation({
        mutationFn: (request: Parameters<typeof importBackup>[0]) => importBackup(request),
        onSuccess: () => {
            client.clear();
            void client.invalidateQueries();
        },
    });
}
