import type { Sort } from "../lib/paging.js";
import { clampLimit } from "./call.js";
import { sortSegment } from "./sorts.js";
import type {
    ClassificationPagesFilter,
    ConversationFilter,
    EdgeFilter,
    EntityFilter,
    ExplainRequest,
    FindingCountsRequest,
    FindingFilter,
    MessageFilter,
    ModelCallFilter,
    PageCountRequest,
    PageFilter,
    PageSelection,
    PendingActionFilter,
    RunFilter,
    ScheduleFilter,
    SiteFilter,
    SiteMapFilter,
    TemplateFilter,
    WriterDirections,
} from "./types.js";

const scope = "pc";

function limitSegment(limit: number | undefined): string {
    return String(clampLimit(limit));
}

export interface UsageScope {
    runId?: string;
    conversationId?: string;
}

export const keys = {
    health: {
        ping: () => [scope, "health", "ping"] as const,
    },
    settings: {
        root: () => [scope, "settings"] as const,
        schema: () => [scope, "settings", "schema"] as const,
        lock: () => [scope, "settings", "lock"] as const,
        values: () => [scope, "settings", "value"] as const,
        value: (key: string) => [scope, "settings", "value", key] as const,
    },
    tools: {
        list: () => [scope, "tools", "list"] as const,
    },
    browser: {
        locate: () => [scope, "browser", "locate"] as const,
    },
    sites: {
        root: () => [scope, "sites"] as const,
        lists: () => [scope, "sites", "list"] as const,
        list: (filter: SiteFilter, sort: Sort | null, limit?: number) =>
            [scope, "sites", "list", filter, sortSegment(sort), limitSegment(limit)] as const,
        details: () => [scope, "sites", "detail"] as const,
        detail: (id: string) => [scope, "sites", "detail", id] as const,
        typesAll: () => [scope, "sites", "types"] as const,
        types: (siteId: string) => [scope, "sites", "types", siteId] as const,
    },
    pages: {
        root: () => [scope, "pages"] as const,
        lists: () => [scope, "pages", "list"] as const,
        list: (filter: PageFilter, sort: Sort | null, limit?: number) =>
            [scope, "pages", "list", filter, sortSegment(sort), limitSegment(limit)] as const,
        details: () => [scope, "pages", "detail"] as const,
        detail: (id: string) => [scope, "pages", "detail", id] as const,
        trees: () => [scope, "pages", "tree"] as const,
        tree: (siteId: string) => [scope, "pages", "tree", siteId] as const,
        models: () => [scope, "pages", "model"] as const,
        model: (pageId: string) => [scope, "pages", "model", pageId] as const,
        summaries: () => [scope, "pages", "summary"] as const,
        summary: (siteId: string) => [scope, "pages", "summary", siteId] as const,
        packs: () => [scope, "pages", "pack"] as const,
        pack: (pageId: string, field: string, directions: WriterDirections | null = null) =>
            [scope, "pages", "pack", pageId, field, directions] as const,
        maps: () => [scope, "pages", "map"] as const,
        map: (filter: SiteMapFilter, limit?: number) => [scope, "pages", "map", filter, limitSegment(limit)] as const,
        counts: () => [scope, "pages", "count"] as const,
        count: (request: PageCountRequest) => [scope, "pages", "count", request] as const,
        selections: () => [scope, "pages", "selection"] as const,
        selection: (request: PageSelection) => [scope, "pages", "selection", request] as const,
    },
    findings: {
        root: () => [scope, "findings"] as const,
        lists: () => [scope, "findings", "list"] as const,
        list: (filter: FindingFilter, limit?: number) => [scope, "findings", "list", filter, limitSegment(limit)] as const,
        ids: (filter: FindingFilter) => [scope, "findings", "ids", filter] as const,
        details: () => [scope, "findings", "detail"] as const,
        detail: (id: string) => [scope, "findings", "detail", id] as const,
        countsAll: () => [scope, "findings", "counts"] as const,
        counts: (request: FindingCountsRequest) => [scope, "findings", "counts", request] as const,
        explanations: () => [scope, "findings", "explain"] as const,
        explain: (request: ExplainRequest) => [scope, "findings", "explain", request] as const,
    },
    previews: {
        root: () => [scope, "preview"] as const,
        link: (pageId: string, status: string) => [scope, "preview", pageId, status] as const,
    },
    graph: {
        root: () => [scope, "graph"] as const,
        entityLists: () => [scope, "graph", "entities"] as const,
        entities: (filter: EntityFilter, sort: Sort | null, limit?: number) =>
            [scope, "graph", "entities", filter, sortSegment(sort), limitSegment(limit)] as const,
        entityAll: () => [scope, "graph", "entity"] as const,
        entity: (id: string) => [scope, "graph", "entity", id] as const,
        edgeLists: () => [scope, "graph", "edges"] as const,
        edges: (filter: EdgeFilter, limit?: number) =>
            [scope, "graph", "edges", filter, limitSegment(limit)] as const,
        fulls: () => [scope, "graph", "full"] as const,
        full: (siteId: string) => [scope, "graph", "full", siteId] as const,
        cardLists: () => [scope, "graph", "cards"] as const,
        cards: (siteId: string) => [scope, "graph", "cards", siteId] as const,
        adoption: (siteId: string, under: string) => [scope, "graph", "adoption", siteId, under] as const,
    },
    classification: {
        root: () => [scope, "classification"] as const,
        rules: (siteId: string) => [scope, "classification", "rules", siteId] as const,
        previews: () => [scope, "classification", "preview"] as const,
        preview: (siteId: string) => [scope, "classification", "preview", siteId] as const,
        pages: (filter: ClassificationPagesFilter, limit?: number) =>
            [scope, "classification", "preview", filter.siteId, "pages", filter, limitSegment(limit)] as const,
        tests: () => [scope, "classification", "test"] as const,
        test: (request: unknown) => [scope, "classification", "test", request] as const,
    },
    templates: {
        root: () => [scope, "templates"] as const,
        lists: () => [scope, "templates", "list"] as const,
        list: (filter: TemplateFilter, sort: Sort | null, limit?: number) =>
            [scope, "templates", "list", filter, sortSegment(sort), limitSegment(limit)] as const,
        details: () => [scope, "templates", "detail"] as const,
        detail: (id: string) => [scope, "templates", "detail", id] as const,
        resolvedAll: () => [scope, "templates", "resolved"] as const,
        resolved: (pageId: string) => [scope, "templates", "resolved", pageId] as const,
        resolvedWith: (pageId: string, templateId: string) =>
            [scope, "templates", "resolved", pageId, templateId] as const,
    },
    runs: {
        root: () => [scope, "runs"] as const,
        lists: () => [scope, "runs", "list"] as const,
        list: (filter: RunFilter, sort: Sort | null, limit?: number) =>
            [scope, "runs", "list", filter, sortSegment(sort), limitSegment(limit)] as const,
        details: () => [scope, "runs", "detail"] as const,
        detail: (runId: string) => [scope, "runs", "detail", runId] as const,
        itemLists: () => [scope, "runs", "items"] as const,
        itemsOf: (runId: string) => [scope, "runs", "items", runId] as const,
        items: (runId: string, status: string | undefined, limit?: number) =>
            [scope, "runs", "items", runId, status ?? "any", limitSegment(limit)] as const,
        artifactsOf: (itemId: string) => [scope, "runs", "artifact", itemId] as const,
        artifact: (itemId: string, kind: string) => [scope, "runs", "artifact", itemId, kind] as const,
        parentsAll: () => [scope, "runs", "parents"] as const,
        parents: (siteId: string, pageIds: readonly string[]) => [scope, "runs", "parents", { siteId, pageIds }] as const,
    },
    schedules: {
        root: () => [scope, "schedules"] as const,
        lists: () => [scope, "schedules", "list"] as const,
        list: (filter: ScheduleFilter, limit?: number) =>
            [scope, "schedules", "list", filter, limitSegment(limit)] as const,
        details: () => [scope, "schedules", "detail"] as const,
        detail: (id: string) => [scope, "schedules", "detail", id] as const,
    },
    agent: {
        root: () => [scope, "agent"] as const,
        conversationsAll: () => [scope, "agent", "conversations"] as const,
        conversations: (filter: ConversationFilter, limit?: number) =>
            [scope, "agent", "conversations", filter, limitSegment(limit)] as const,
        messagesAll: () => [scope, "agent", "messages"] as const,
        messagesOf: (conversationId: string) => [scope, "agent", "messages", { conversationId }] as const,
        messages: (filter: MessageFilter, limit?: number) =>
            [scope, "agent", "messages", filter, limitSegment(limit)] as const,
        pendingAll: () => [scope, "agent", "pending"] as const,
        pending: (filter: PendingActionFilter, limit?: number) =>
            [scope, "agent", "pending", filter, limitSegment(limit)] as const,
        plans: (conversationId: string) => [scope, "agent", "plans", conversationId] as const,
    },
    models: {
        root: () => [scope, "models"] as const,
        catalog: () => [scope, "models", "catalog"] as const,
        providerKeys: () => [scope, "models", "providerKeys"] as const,
        profilesAll: () => [scope, "models", "profiles"] as const,
        profiles: (siteId: string | undefined) => [scope, "models", "profiles", siteId ?? "global"] as const,
        usageAll: () => [scope, "models", "usage"] as const,
        usage: (usageScope: UsageScope) => [scope, "models", "usage", usageScope] as const,
        spendAll: () => [scope, "models", "spend"] as const,
        spendRanges: () => [scope, "models", "spend", "days"] as const,
        spendOver: (days: number) => [scope, "models", "spend", "days", days] as const,
        spendOfRun: (runId: string) => [scope, "models", "spend", "run", runId] as const,
        callLists: () => [scope, "models", "calls"] as const,
        calls: (filter: ModelCallFilter, limit?: number) =>
            [scope, "models", "calls", filter, limitSegment(limit)] as const,
    },
    reports: {
        root: () => [scope, "reports"] as const,
        sites: () => [scope, "reports", "site"] as const,
        site: (siteId: string) => [scope, "reports", "site", siteId] as const,
        siteLinks: (siteId: string) => [scope, "reports", "site", siteId, "links"] as const,
        pages: () => [scope, "reports", "page"] as const,
        page: (pageId: string) => [scope, "reports", "page", pageId] as const,
        pageLinks: (pageId: string) => [scope, "reports", "page", pageId, "links"] as const,
        runsAll: () => [scope, "reports", "run"] as const,
        run: (runId: string) => [scope, "reports", "run", runId] as const,
        judge: (pageId: string) => [scope, "reports", "judge", pageId] as const,
    },
    imports: {
        root: () => [scope, "imports"] as const,
        mappingsAll: () => [scope, "imports", "mappings"] as const,
        mappings: (siteId: string) => [scope, "imports", "mappings", siteId] as const,
        inspect: (siteId: string, path: string) => [scope, "imports", "inspect", siteId, path] as const,
        preview: (siteId: string, path: string, mappingId: string) =>
            [scope, "imports", "preview", siteId, path, mappingId] as const,
    },
    sync: {
        root: () => [scope, "sync"] as const,
        plugins: () => [scope, "sync", "plugin"] as const,
        plugin: (siteId: string) => [scope, "sync", "plugin", siteId] as const,
    },
};

export function siteOf(key: readonly unknown[]): string | null {
    const filters = key[3];
    if (typeof filters !== "object" || filters === null) {
        return null;
    }
    const held = (filters as { siteId?: unknown }).siteId;
    return typeof held === "string" && held !== "" ? held : null;
}
