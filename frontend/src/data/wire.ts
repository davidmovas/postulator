export type Timestamp = string | null;

type IsAny<T> = 0 extends 1 & T ? true : false;

type TimeField =
    | "at"
    | "createdAt"
    | "updatedAt"
    | "startedAt"
    | "finishedAt"
    | "deadlineAt"
    | "expiresAt"
    | "wakeAt"
    | "observedAt"
    | "wpModifiedAt"
    | "lastSyncedAt"
    | "nextRunAt";

export type Wire<T> =
    IsAny<T> extends true
        ? T
        : T extends readonly (infer E)[]
          ? Wire<E>[]
          : T extends (...args: never[]) => unknown
            ? T
            : T extends object
              ? { [K in keyof T]: K extends TimeField ? Timestamp : Wire<T[K]> }
              : T;
