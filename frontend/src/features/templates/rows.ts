import { useMemo } from "react";

import { flatten } from "../../data/call.js";
import { useTemplates } from "../../data/hooks/templates.js";
import type { Template } from "../../data/types.js";
import type { TemplateSort } from "../../data/sorts.js";
import type { TemplatesQuery } from "./params.js";
import { sorts } from "./params.js";

const pageSize = 100;

export function matches(template: Template, search: string): boolean {
    const needle = search.trim().toLowerCase();
    if (needle === "") {
        return true;
    }
    return template.name.toLowerCase().includes(needle) || template.pageKind.toLowerCase().includes(needle);
}

export function ordered(rows: readonly Template[], sort: TemplateSort): readonly Template[] {
    const out = [...rows];
    out.sort((a, b) => {
        const held =
            sort.field === "name"
                ? a.name.localeCompare(b.name, undefined, { sensitivity: "base" })
                : (a.createdAt ?? "").localeCompare(b.createdAt ?? "");
        return sort.desc ? -held : held;
    });
    return out;
}

export function filtered(rows: readonly Template[], query: TemplatesQuery): readonly Template[] {
    return rows.filter((template) => {
        if (query.scope !== "all" && template.scope !== query.scope) {
            return false;
        }
        if (query.pageKind !== "" && template.pageKind !== query.pageKind) {
            return false;
        }
        return matches(template, query.search);
    });
}

export function kindsOf(rows: readonly Template[]): readonly string[] {
    return [...new Set(rows.map((template) => template.pageKind).filter((kind) => kind !== ""))].sort();
}

export function namesIn(rows: readonly Template[], scope: string): readonly string[] {
    return rows.filter((template) => template.scope === scope).map((template) => template.name);
}

export interface TemplateRows {
    shown: readonly Template[];
    loaded: readonly Template[];
    kinds: readonly string[];
    pending: boolean;
    error: unknown;
    hasMore: boolean;
    loadingMore: boolean;
    loadMore: () => void;
    retry: () => void;
}

export function useTemplateRows(siteId: string, query: TemplatesQuery): TemplateRows {
    const sort = query.sort ?? sorts.nameAsc;
    const globals = useTemplates({ scope: "global" }, sort, pageSize);
    const locals = useTemplates({ siteId }, sort, pageSize);
    const globalRows = useMemo(() => flatten(globals.data?.pages), [globals.data]);
    const localRows = useMemo(() => flatten(locals.data?.pages), [locals.data]);
    const loaded = useMemo(() => [...globalRows, ...localRows], [globalRows, localRows]);
    const shown = useMemo(() => ordered(filtered(loaded, query), sort), [loaded, query, sort]);
    const kinds = useMemo(() => kindsOf(loaded), [loaded]);

    return {
        shown,
        loaded,
        kinds,
        pending: globals.isPending || locals.isPending,
        error: globals.error ?? locals.error,
        hasMore: globals.hasNextPage || locals.hasNextPage,
        loadingMore: globals.isFetchingNextPage || locals.isFetchingNextPage,
        loadMore: () => {
            if (globals.hasNextPage) {
                void globals.fetchNextPage();
            }
            if (locals.hasNextPage) {
                void locals.fetchNextPage();
            }
        },
        retry: () => {
            void globals.refetch();
            void locals.refetch();
        },
    };
}
