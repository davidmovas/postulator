export type ScheduleType =
  | "manual"
  | "once"
  | "interval"
  | "daily"
  | "weekly"
  | "monthly";

export const ScheduleTypeConst = {
  Manual: "manual" as ScheduleType,
  Once: "once" as ScheduleType,
  Interval: "interval" as ScheduleType,
  Daily: "daily" as ScheduleType,
  Weekly: "weekly" as ScheduleType,
  Monthly: "monthly" as ScheduleType,
} as const;

export type JobStatus = "active" | "paused" | "completed" | "error";

export const JobStatusConst = {
  Active: "active" as JobStatus,
  Paused: "paused" as JobStatus,
  Completed: "completed" as JobStatus,
  Error: "error" as JobStatus,
} as const;


export type ExecutionStatus =
  | "pending"
  | "generating"
  | "pending_validation"
  | "validated"
  | "publishing"
  | "published"
  | "failed";

export const ExecutionStatusConst = {
  Pending: "pending" as ExecutionStatus,
  Generating: "generating" as ExecutionStatus,
  PendingValidation: "pending_validation" as ExecutionStatus,
  Validated: "validated" as ExecutionStatus,
  Publishing: "publishing" as ExecutionStatus,
  Published: "published" as ExecutionStatus,
  Failed: "failed" as ExecutionStatus,
} as const;
