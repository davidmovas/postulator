export const categories = {
    trail: "WordPress categories",
    onSite: (termId: number) => `WordPress category #${termId}`,
    createdByRun: (termId: number) => `Created on WordPress by this run as category #${termId}`,
    onPublish: "Created on WordPress when a page under it is published",
    needsPlugin: "Pages carry categories only with the companion plugin 1.3.0 — update it and sync",
    becomesHint: "A new category from this sheet. WordPress gets it when a page under it is published.",
    known: "A category Postulator already holds",
};
