import { Templates } from "../../lib/api.js";
import { listed, one } from "../call.js";
import type { LinkPolicy, Template } from "../types.js";

export const createTemplate = one(Templates.CreateTemplate);
export const updateTemplate = one(Templates.UpdateTemplate);
export const deleteTemplate = one(Templates.DeleteTemplate);
export const getTemplate = one(Templates.GetTemplate);
export const listTemplates = listed<Parameters<typeof Templates.ListTemplates>[0], Template>(
    Templates.ListTemplates,
);
export const setOverride = one(Templates.SetOverride);
export const deleteOverride = one(Templates.DeleteOverride);
export const resolveForPage = one(Templates.ResolveForPage);
export const createPolicy = one(Templates.CreatePolicy);
export const updatePolicy = one(Templates.UpdatePolicy);
export const deletePolicy = one(Templates.DeletePolicy);
export const getPolicy = one(Templates.GetPolicy);
export const listPolicies = listed<Parameters<typeof Templates.ListPolicies>[0], LinkPolicy>(
    Templates.ListPolicies,
);
export const getEffectivePolicy = one(Templates.GetEffectivePolicy);
