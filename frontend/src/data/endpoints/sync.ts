import { Sync } from "../../lib/api.js";
import { one } from "../call.js";

export const syncSite = one(Sync.SyncSite);
export const checkPlugin = one(Sync.CheckPlugin);
export const savePluginPackage = one(Sync.SavePluginPackage);
