import { Schedules } from "../../lib/api.js";
import { listed, one } from "../call.js";
import type { Schedule } from "../types.js";

export const createSchedule = one(Schedules.Create);
export const updateSchedule = one(Schedules.Update);
export const deleteSchedule = one(Schedules.Delete);
export const getSchedule = one(Schedules.Get);
export const listSchedules = listed<Parameters<typeof Schedules.List>[0], Schedule>(Schedules.List);
export const enableSchedule = one(Schedules.Enable);
export const disableSchedule = one(Schedules.Disable);
export const runScheduleNow = one(Schedules.RunNow);
