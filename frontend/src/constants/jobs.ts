// Shared job-related constants and types aligned with Go backend enums

// ScheduleType mirrors Go:
// type ScheduleType string
// const (
//   ScheduleManual  ScheduleType = "manual"
//   ScheduleOnce    ScheduleType = "once"
//   ScheduleInterval ScheduleType = "interval"
// )
export type ScheduleType =
  | "manual"
  | "once"
  | "interval";

export const ScheduleTypeConst = {
  Manual: "manual" as ScheduleType,
  Once: "once" as ScheduleType,
  Interval: "interval" as ScheduleType,
} as const;

// Job Status mirrors Go:
// type Status string
// const (
//   StatusActive    Status = "active"
//   StatusPaused    Status = "paused"
//   StatusCompleted Status = "completed"
//   StatusError     Status = "error"
// )
export type JobStatus = "active" | "paused" | "completed" | "error";

export const JobStatusConst = {
  Active: "active" as JobStatus,
  Paused: "paused" as JobStatus,
  Completed: "completed" as JobStatus,
  Error: "error" as JobStatus,
} as const;

// ExecutionStatus mirrors Go:
// type ExecutionStatus string
// const (
//   ExecutionPending           ExecutionStatus = "pending"
//   ExecutionGenerating        ExecutionStatus = "generating"
//   ExecutionPendingValidation ExecutionStatus = "pending_validation"
//   ExecutionValidated         ExecutionStatus = "validated"
//   ExecutionPublishing        ExecutionStatus = "publishing"
//   ExecutionPublished         ExecutionStatus = "published"
//   ExecutionFailed            ExecutionStatus = "failed"
// )
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
