import { Reports } from "../../lib/api.js";
import { one } from "../call.js";

export const siteOverview = one(Reports.SiteOverview);
export const pageReport = one(Reports.PageReport);
export const runReport = one(Reports.RunReport);
export const judgePage = one(Reports.JudgePage);
