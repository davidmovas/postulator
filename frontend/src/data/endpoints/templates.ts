import { Templates } from "../../lib/api.js";
import { listed, one } from "../call.js";
import type { ClassificationPage, Template } from "../types.js";

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
export const listPageTypes = one(Templates.ListPageTypes);
export const createPageType = one(Templates.CreatePageType);
export const savePageType = one(Templates.SavePageType);
export const deletePageType = one(Templates.DeletePageType);
export const setScope = one(Templates.SetScope);
export const listRules = one(Templates.ListRules);
export const saveRule = one(Templates.SaveRule);
export const deleteRule = one(Templates.DeleteRule);
export const reorderRules = one(Templates.ReorderRules);
export const testRule = one(Templates.TestRule);
export const previewClassification = one(Templates.PreviewClassification);
export const previewClassificationPages = listed<Parameters<typeof Templates.PreviewClassificationPages>[0], ClassificationPage>(
    Templates.PreviewClassificationPages,
);
export const applyClassification = one(Templates.ApplyClassification);
