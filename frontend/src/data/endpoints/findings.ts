import { Findings } from "../../lib/api.js";
import { listed, one } from "../call.js";
import type { Finding } from "../types.js";

export const listFindings = listed<Parameters<typeof Findings.List>[0], Finding>(Findings.List);
export const getFinding = one(Findings.Get);
export const countFindings = one(Findings.Counts);
export const explainResolution = one(Findings.Explain);
export const resolveFinding = one(Findings.Resolve);
export const resolveFindings = one(Findings.ResolveMany);
