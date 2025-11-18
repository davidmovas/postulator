export const ALLOWED_IMPORT_EXTENSIONS = ["txt", "csv", "json", "xlsx"] as const;
export type AllowedImportExtension = typeof ALLOWED_IMPORT_EXTENSIONS[number];

export const TOPIC_STRATEGY_UNIQUE = "unique";
export const TOPIC_STRATEGY_REUSE_WITH_VARIATION = "reuse_with_variation";

export const TOPIC_STRATEGIES = [TOPIC_STRATEGY_UNIQUE, TOPIC_STRATEGY_REUSE_WITH_VARIATION] as const;
export type TopicStrategy = typeof TOPIC_STRATEGIES[number];
export const DEFAULT_TOPIC_STRATEGY: TopicStrategy = "unique";
