import { Import } from "../../lib/api.js";
import { one } from "../call.js";

export const inspectSheet = one(Import.Inspect);
export const previewSheet = one(Import.Preview);
export const applySheet = one(Import.Apply);
export const exportSite = one(Import.Export);
export const saveMapping = one(Import.SaveMapping);
export const listMappings = one(Import.ListMappings);
export const deleteMapping = one(Import.DeleteMapping);
