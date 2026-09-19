import { Sites } from "../../lib/api.js";
import { listed, one } from "../call.js";
import type { Site } from "../types.js";

export const createSite = one(Sites.Create);
export const updateSite = one(Sites.Update);
export const deleteSite = one(Sites.Delete);
export const getSite = one(Sites.Get);
export const listSites = listed<Parameters<typeof Sites.List>[0], Site>(Sites.List);
