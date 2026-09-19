import { Settings } from "../../lib/api.js";
import { one } from "../call.js";

export const settingsSchema = one(Settings.Schema);
export const getSetting = one(Settings.Get);
export const setSetting = one(Settings.Set);
export const setProviderKey = one(Settings.SetProviderKey);
export const lockState = one(Settings.LockState);
export const lock = one(Settings.Lock);
export const unlock = one(Settings.Unlock);
export const setMasterPassword = one(Settings.SetMasterPassword);
export const exportBackup = one(Settings.ExportBackup);
export const importBackup = one(Settings.ImportBackup);
