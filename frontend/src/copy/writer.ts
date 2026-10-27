export const writer = {
    title: "What the writer will see",
    loading: "Building what the writer will see",
    field: "Field",
    fieldHint:
        "Each field is written from its own picture of the site. The body, or a product's description, is the one that carries the links.",
    summaryTitle: "In one call",
    kept: (kept: string, built: string) => `${kept} of ${built} tokens go to the model`,
    withinBudget: (budget: string) => `The input budget is ${budget} tokens.`,
    wholeWindow: "No input budget is set, so the call may fill the model's whole window.",
    budgetHint: "The budget per role is set in Settings, Models.",
    openSettings: "Open Settings, Models",
    sectionsTitle: "Section by section",
    priority: (rank: number) => `priority ${rank}`,
    priorityHint: "A tight budget cuts the highest number first, a list from its end. Priority 1 is never cut.",
    tokens: (count: string) => `${count} tokens`,
    noSections: "The writer would see nothing about this page.",
    dropped: {
        title: "Left out for the budget",
        none: "Nothing was left out.",
        whole: "the whole section",
        items: (count: number) => (count === 1 ? "1 item from the end" : `${count} items from the end`),
    },
    sectionTitles: {
        brief: "The run's brief and this page's directions",
        reference: "Reference text given for the run",
        style: "Style sample given for the run",
    } as Readonly<Record<string, string>>,
    directed: {
        draft: "With the brief and this page's directions as the run start holds them now.",
        run: "With this run's brief and this page's directions, over the model as it stands now.",
    },
    drawer: {
        draft: (path: string) => `What the writer will see for ${path}`,
        run: (path: string) => `What this run's writer reads for ${path}`,
        loading: "Reading the page",
        close: "Close",
    },
    sawRun: "What the writer saw",
    tooSmall: (budget: string, needed: string) =>
        `The input budget of ${budget} tokens is smaller than the ${needed} every call about this page must carry, so a run would be refused before it starts. Raise the budget, or set it to 0 for the model's whole window.`,
};
