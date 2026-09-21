import { Browser } from "../../lib/api.js";
import { one } from "../call.js";

export const openBrowser = one(Browser.Open);
export const locateBrowser = one(Browser.Locate);
