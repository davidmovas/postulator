export const ALLOWED_IMPORT_EXTENSIONS = ["txt", "csv", "json", "xlsx"] as const;
export type AllowedImportExtension = typeof ALLOWED_IMPORT_EXTENSIONS[number];

export const TOPIC_STRATEGIES = ["unique", "variation"] as const;
export type TopicStrategy = typeof TOPIC_STRATEGIES[number];
export const DEFAULT_TOPIC_STRATEGY: TopicStrategy = "unique";
