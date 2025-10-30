import { dto } from "@/wailsjs/wailsjs/go/models";

export class ServiceError extends Error {
  code?: string;
  details?: any;
  constructor(message: string, code?: string, details?: any) {
    super(message);
    this.name = "ServiceError";
    this.code = code;
    this.details = details;
  }
}

export function unwrapOne<T>(res: { success: boolean; data?: T; error?: dto.Error }, onEmpty?: () => never): T {
  if (!res) throw new ServiceError("Empty response from backend");
  if (!res.success) throw new ServiceError(res.error?.message || "Request failed", res.error?.code, res.error);
  if (res.data == null) {
    if (onEmpty) return onEmpty();
    throw new ServiceError("No data returned");
  }
  return res.data as T;
}

export function unwrapMany<T>(res: { success: boolean; data?: T[]; error?: dto.Error }): T[] {
  if (!res) throw new ServiceError("Empty response from backend");
  if (!res.success) throw new ServiceError(res.error?.message || "Request failed", res.error?.code, res.error);
  return (res.data ?? []) as T[];
}

export function unwrapString(res: dto.Response_string_): string {
  if (!res) throw new ServiceError("Empty response from backend");
  if (!res.success) throw new ServiceError(res.error?.message || "Request failed", res.error?.code, res.error);
  return res.data ?? "";
}
