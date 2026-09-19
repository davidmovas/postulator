import { Pages } from "../../lib/api.js";
import { listed, one } from "../call.js";
import type { Page } from "../types.js";

export const createPage = one(Pages.Create);
export const updatePage = one(Pages.Update);
export const deletePage = one(Pages.Delete);
export const getPage = one(Pages.Get);
export const listPages = listed<Parameters<typeof Pages.List>[0], Page>(Pages.List);
export const pageTree = one(Pages.Tree);
export const mapPageToEntity = one(Pages.MapToEntity);
export const unmapPage = one(Pages.Unmap);
export const setCanonicalPage = one(Pages.SetCanonical);
export const replacePageLinks = one(Pages.ReplaceLinks);
export const previewLink = one(Pages.PreviewLink);
