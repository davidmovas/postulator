import { isOneOf } from "../generated/vocab.js";
import type { SortOf } from "./sorts.js";
import { formatSort, parseSort } from "./sorts.js";

export interface Param<T> {
    readonly key: string;
    read(raw: string | null): T;
    write(value: T): string;
}

export function textParam(key: string): Param<string> {
    return {
        key,
        read: (raw) => raw ?? "",
        write: (value) => value,
    };
}

export function flagParam(key: string): Param<boolean> {
    return {
        key,
        read: (raw) => raw === "1",
        write: (value) => (value ? "1" : ""),
    };
}

export function choiceParam<V extends string, F extends string>(
    key: string,
    values: readonly V[],
    fallback: F,
): Param<V | F> {
    return {
        key,
        read: (raw) => (raw !== null && isOneOf(values, raw) ? raw : fallback),
        write: (value) => (value === fallback ? "" : value),
    };
}

export function listParam<V extends string>(key: string, values: readonly V[]): Param<readonly V[]> {
    return {
        key,
        read: (raw) =>
            (raw ?? "")
                .split(",")
                .map((held) => held.trim())
                .filter((held): held is V => isOneOf(values, held)),
        write: (value) => value.join(","),
    };
}

export function sortParam<F extends string>(key: string, fields: readonly F[]): Param<SortOf<readonly F[]> | null> {
    return {
        key,
        read: (raw) => parseSort(fields, raw),
        write: (value) => formatSort(value),
    };
}

export type Fields<Q> = { readonly [K in keyof Q]: Param<Q[K]> };

export interface QueryCodec<Q> {
    readonly defaults: Q;
    read: (params: URLSearchParams) => Q;
    write: (query: Q) => URLSearchParams;
    search: (query: Q) => string;
    carries: (query: Q, keys: readonly (keyof Q)[]) => boolean;
}

export function queryCodec<Q extends object>(fields: Fields<Q>): QueryCodec<Q> {
    const keys = Object.keys(fields) as (keyof Q)[];
    const read = (params: URLSearchParams): Q => {
        const query: Partial<Q> = {};
        for (const key of keys) {
            const param = fields[key];
            query[key] = param.read(params.get(param.key));
        }
        return query as Q;
    };
    const write = (query: Q): URLSearchParams => {
        const params = new URLSearchParams();
        for (const key of keys) {
            const param = fields[key];
            const written = param.write(query[key]);
            if (written !== "") {
                params.set(param.key, written);
            }
        }
        return params;
    };
    return {
        defaults: read(new URLSearchParams()),
        read,
        write,
        search: (query) => {
            const serialised = write(query).toString();
            return serialised === "" ? "" : `?${serialised}`;
        },
        carries: (query, named) => named.some((key) => fields[key].write(query[key]) !== ""),
    };
}

export const actionParam = "action";

export const actionNew = "new";

export function wantsNew(params: URLSearchParams): boolean {
    return params.get(actionParam) === actionNew;
}
