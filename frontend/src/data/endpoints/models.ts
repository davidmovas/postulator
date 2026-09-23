import { Models } from "../../lib/api.js";
import { one } from "../call.js";

export const listModels = one(Models.ListModels);
export const upsertModel = one(Models.UpsertModel);
export const disableModel = one(Models.DisableModel);
export const getProfiles = one(Models.GetProfiles);
export const setProfile = one(Models.SetProfile);
export const testProvider = one(Models.TestProvider);
export const usageSummary = one(Models.UsageSummary);
