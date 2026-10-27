interface WhyFacts {
    crumb?: string;
    candidates?: readonly string[] | null;
    loop?: readonly string[] | null;
    parent?: string;
}

const quoted = (text: string): string => `“${text}”`;

export const map = {
    why: {
        none: "No parent",
        labels: {
            root: "Front page",
            parent_archived: "Parent left the site",
            cycle_broken: "Loop broken here",
            crumb_ambiguous: "Crumb matches several",
            crumb_unresolved: "Crumb leads nowhere",
            planned: "Planned, not on the site",
            render_failed: "Page not fetched",
            not_derived: "Not read yet",
            no_signal: "Nothing places it",
        },
        sentences: {
            root: () => "The front page is the root of the site, so nothing sits above it.",
            parent_archived: ({ parent }: WhyFacts) =>
                parent === undefined || parent === ""
                    ? "Its parent was taken off the site, so it sits at the top."
                    : `Its parent ${parent} was taken off the site, so it sits at the top.`,
            cycle_broken: ({ loop }: WhyFacts) =>
                (loop ?? []).length === 0
                    ? "Its parents looped back on themselves, so the loop was broken here."
                    : `Its parents looped through ${(loop ?? []).join(" › ")}, so the loop was broken here.`,
            crumb_ambiguous: ({ crumb, candidates }: WhyFacts) => {
                const named = crumb === undefined || crumb === "" ? "a name" : quoted(crumb);
                const listed = candidates ?? [];
                return listed.length === 0
                    ? `Its breadcrumb names ${named}, which several pages carry, so none was taken.`
                    : `Its breadcrumb names ${named}, which ${listed.length} pages carry: ${listed.join(", ")}. None was taken.`;
            },
            crumb_unresolved: ({ crumb }: WhyFacts) =>
                crumb === undefined || crumb === ""
                    ? "Its breadcrumb leads to no page in the model."
                    : `Its breadcrumb points at ${crumb}, which is no page in the model.`,
            planned: () => "It is planned and not on the site yet, so no breadcrumb places it.",
            render_failed: () => "The page could not be fetched from the site, so its breadcrumb was not read.",
            not_derived: () => "The page has not been read for a breadcrumb yet; the next sync reads it.",
            no_signal: () =>
                "Nothing places it under another page: no breadcrumb, no WordPress parent and no parent in its address.",
        },
        finding: "Open its finding",
    },
    title: "Site map",
    loading: "Reading the site map",
    loadingBranch: "Reading the pages below",
    more: "Show more",
    columns: {
        page: "Page",
        type: "Type",
        parent: "Parent from",
        body: "Body",
        links: "In · out",
        findings: "Findings",
        children: "Below",
    },
    untyped: "No type",
    untitled: "Untitled",
    expand: (path: string) => `Show the pages below ${path}`,
    collapse: (path: string) => `Hide the pages below ${path}`,
    open: "Open the page",
    twin: (path: string) => `with ${path}`,
    twinTooltip: "The hub and the store category of the same name are one node to the writer.",
    technical: "Technical",
    readOnly: "Read only",
    trail: (names: readonly string[]) => `in ${names.join(" › ")}`,
    links: (incoming: number, outgoing: number) => `${incoming} · ${outgoing}`,
    linksTooltip: "Contextual links that point to this page, and that it carries.",
    findingsTooltip: "Open the findings of this page and every page below it.",
    childrenTooltip: (children: number, descendants: number) =>
        `${children === 1 ? "1 page" : `${children} pages`} directly below, ${descendants} in all.`,
    under: "Show only under this page",
    whole: "Whole site",
    subtree: "Subtree",
    search: "Find a page",
    searchHint: "Title or path",
    type: "Page type",
    everyType: "Every type",
    bodyStates: {
        empty: "Empty",
        client: "Client's",
        ours: "Ours",
        mixed: "Mixed",
        dynamic: "Dynamic",
    },
    bodyTooltips: {
        empty: "The body holds nothing yet.",
        client: "Every block of the body is the client's own.",
        ours: "Every block of the body was written by a run.",
        mixed: "The body holds the client's blocks and blocks a run wrote.",
        dynamic: "The site builds the body, so it is read and never written.",
    },
    empty: {
        title: "No pages in the model yet",
        body: "A sync reads the site's pages into the model, and an import plans them from a workbook.",
        sync: "Sync now",
        syncing: "Syncing",
        import: "Import",
    },
    noMatch: {
        title: "No page matches",
        reset: "Show the whole tree",
    },
    noChildren: "Nothing sits below this page.",
};
