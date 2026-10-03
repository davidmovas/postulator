import { Models } from "../../lib/api.js";
import { listed, one } from "../call.js";
import type { ModelCall } from "../types.js";

export const listModels = one(Models.ListModels);
export const upsertModel = one(Models.UpsertModel);
export const disableModel = one(Models.DisableModel);
export const getProfiles = one(Models.GetProfiles);
export const setProfile = one(Models.SetProfile);
export const testProvider = one(Models.TestProvider);
export const usageSummary = one(Models.UsageSummary);
export const spendReport = one(Models.SpendReport);
export const listCalls = listed<Parameters<typeof Models.ListCalls>[0], ModelCall>(Models.ListCalls);
