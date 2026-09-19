import { Runs } from "../../lib/api.js";
import { listed, one } from "../call.js";
import type { Run, RunItem } from "../types.js";

export const startRun = one(Runs.Start);
export const estimateRun = one(Runs.Estimate);
export const getRun = one(Runs.Get);
export const listRuns = listed<Parameters<typeof Runs.List>[0], Run>(Runs.List);
export const listRunItems = listed<Parameters<typeof Runs.ListItems>[0], RunItem>(Runs.ListItems);
export const listRunEvents = one(Runs.ListEvents);
export const getArtifact = one(Runs.GetArtifact);
export const listArtifacts = one(Runs.ListArtifacts);
export const pauseRun = one(Runs.Pause);
export const resumeRun = one(Runs.Resume);
export const cancelRun = one(Runs.Cancel);
export const retryStep = one(Runs.RetryStep);
