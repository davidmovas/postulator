import {
  CreateJob,
  DeleteJob,
  ExecuteJobManually,
  GetJob,
  GetPendingValidations,
  ListJobs,
  PauseJob,
  RestoreScheduler,
  ResumeJob,
  UpdateJob,
  ValidateExecution,
} from "@/wailsjs/wailsjs/go/app/App";
import { dto } from "@/wailsjs/wailsjs/go/models";
import { unwrapMany, unwrapOne, unwrapString } from "./utils";

import type { ScheduleType, JobStatus, ExecutionStatus } from "@/constants/jobs";

export interface Job {
  id: number;
  name: string;
  siteId: number;
  categoryId: number;
  promptId: number;
  aiProviderId: number;
  aiModel: string;
  requiresValidation: boolean;
  scheduleType: ScheduleType;
  intervalValue?: number;
  intervalUnit?: string;
  scheduleHour?: number;
  scheduleMinute?: number;
  weekdays?: number[];
  monthdays?: number[];
  jitterEnabled: boolean;
  jitterMinutes: number;
  status: JobStatus;
  lastRunAt?: string;
  nextRunAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface Execution {
  id: number;
  jobId: number;
  topicId: number;
  generatedTitle?: string;
  generatedContent?: string;
  status: ExecutionStatus;
  errorMessage?: string;
  articleId?: number;
  startedAt: string;
  generatedAt?: string;
  validatedAt?: string;
  publishedAt?: string;
}

function mapJob(x: dto.Job): Job {
  return {
    id: x.id,
    name: x.name,
    siteId: x.siteId,
    categoryId: x.categoryId,
    promptId: x.promptId,
    aiProviderId: x.aiProviderId,
    aiModel: x.aiModel,
    requiresValidation: x.requiresValidation,
    scheduleType: x.scheduleType as ScheduleType,
    intervalValue: x.intervalValue,
    intervalUnit: x.intervalUnit,
    scheduleHour: x.scheduleHour,
    scheduleMinute: x.scheduleMinute,
    weekdays: x.weekdays,
    monthdays: x.monthdays,
    jitterEnabled: x.jitterEnabled,
    jitterMinutes: x.jitterMinutes,
    status: x.status as JobStatus,
    lastRunAt: x.lastRunAt,
    nextRunAt: x.nextRunAt,
    createdAt: x.createdAt,
    updatedAt: x.updatedAt,
  };
}

function mapExecution(x: dto.Execution): Execution {
  return {
    id: x.id,
    jobId: x.jobId,
    topicId: x.topicId,
    generatedTitle: x.generatedTitle,
    generatedContent: x.generatedContent,
    status: x.status as ExecutionStatus,
    errorMessage: x.errorMessage,
    articleId: x.articleId,
    startedAt: x.startedAt,
    generatedAt: x.generatedAt,
    validatedAt: x.validatedAt,
    publishedAt: x.publishedAt,
  };
}

export async function listJobs(): Promise<Job[]> {
  const res = await ListJobs();
  return unwrapMany<dto.Job>(res).map(mapJob);
}

export async function getJob(id: number): Promise<Job> {
  const res = await GetJob(id);
  return mapJob(unwrapOne<dto.Job>(res));
}

export async function createJob(input: Omit<Job, "id" | "createdAt" | "updatedAt" | "lastRunAt" | "nextRunAt" | "status"> & { status?: string }): Promise<string> {
  const payload = new dto.Job({ ...input });
  const res = await CreateJob(payload);
  return unwrapString(res);
}

export async function updateJob(input: Omit<Job, "createdAt" | "updatedAt">): Promise<string> {
  const payload = new dto.Job({ ...input });
  const res = await UpdateJob(payload);
  return unwrapString(res);
}

export async function deleteJob(id: number): Promise<string> {
  const res = await DeleteJob(id);
  return unwrapString(res);
}

export async function executeJobManually(id: number): Promise<string> {
  const res = await ExecuteJobManually(id);
  return unwrapString(res);
}

export async function pauseJob(id: number): Promise<string> {
  const res = await PauseJob(id);
  return unwrapString(res);
}

export async function resumeJob(id: number): Promise<string> {
  const res = await ResumeJob(id);
  return unwrapString(res);
}

export async function restoreScheduler(): Promise<string> {
  const res = await RestoreScheduler();
  return unwrapString(res);
}

export async function getPendingValidations(): Promise<Execution[]> {
  const res = await GetPendingValidations();
  return unwrapMany<dto.Execution>(res).map(mapExecution);
}

export async function validateExecution(executionId: number, approved: boolean): Promise<string> {
  const res = await ValidateExecution(executionId, approved);
  return unwrapString(res);
}
