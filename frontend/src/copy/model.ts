const quoted = (text: string): string => `“${text}”`;

export const model = {
    loading: "Reading the page model",
    notOnSite: "This page is not on the site yet, so nothing has been observed.",
    notObserved: "Not observed yet: the next sync reads this page.",
    identity: {
        title: "On the site",
        address: "Address",
        type: "WordPress type",
        typeWithId: (type: string, id: number) => `${type} · id ${id}`,
        status: "Status",
        modified: "Modified on the site",
        observed: "Last observed",
        terms: "Terms",
    },
    hierarchy: {
        title: "Parent",
        front: "The front page, the root of the site.",
        top: "No parent: the page sits at the top of the site.",
        root: "root of the site",
        reason: (source: string, method: string) => (method === "" ? source : `${source}, ${method}`),
    },
    reasons: {
        title: "Technical and read only",
        technical: "Technical page",
        readOnly: "Read only",
        term: "A category page, read only",
        builder: (name: string) => `Made with ${name}, read only`,
    },
    trails: {
        title: "Breadcrumbs",
        none: "The render carries no breadcrumb.",
        crumbs: (count: number) => (count === 1 ? "1 crumb" : `${count} crumbs`),
        unnamed: "Unnamed crumb",
        self: "this page",
        noLink: "no link",
        redirects: (path: string) => `redirects to ${path}`,
        status: (status: number) => `answered ${status}`,
        unresolved: "no page in the model",
    },
    render: {
        title: "Render",
        none: "No render yet: the next sync visits this page.",
        extracted: (when: string) => `read ${when}`,
        root: "Content root",
        words: "Words",
        wordCounts: (content: number, generated: number) =>
            `${content} of the content · ${generated} generated around it`,
        builder: "Builder",
        urlForm: "Link form",
        noLinks: "No internal links",
        robots: "Robots",
        robotsUnset: "Not set",
        noIndex: "Hidden from search engines",
        landedOn: (path: string) => `The visit landed on ${path}`,
        regions: "Regions",
        inside: "inside the content",
        outside: "outside the content",
        regionCounts: (links: number, words: number) =>
            `${links === 1 ? "1 link" : `${links} links`} · ${words === 1 ? "1 word" : `${words} words`}`,
    },
    links: {
        title: "Links by type",
        none: "No links read yet.",
        count: (total: number, contextual: number) =>
            `${total === 1 ? "1 link" : `${total} links`} · ${contextual} contextual`,
        ours: "Written by a run",
    },
    blocks: {
        title: "Body",
        none: "The body has not been read yet.",
        empty: "The body is empty.",
        dynamic: "dynamic",
        ours: "written by a run",
    },
    fields: {
        title: "Fields a run wrote",
        none: "No run has written a field of this page.",
        written: (when: string) => `run of ${when}`,
        ours: "as the run wrote it",
        changed: "changed on the site since",
        handBack: "A revert hands it back and keeps the client's value.",
        revert: "Revert",
        revertOne: (label: string) => `Revert ${label}`,
        ourBody: (label: string) =>
            `${label} gets the value it had before the run that wrote it; a page that run created goes to the trash whole.`,
        handBackBody: (label: string) =>
            `${label} changed on the site since the run wrote it, so the revert keeps the client's value and hands the field back.`,
    },
    findings: {
        title: "Open findings",
        none: "No open findings.",
        unresolved: (name: string, path: string) => {
            if (name === "") {
                return "A crumb of the breadcrumb leads to no page in the model.";
            }
            return path === ""
                ? `The breadcrumb names ${quoted(name)}, which leads to no page in the model.`
                : `The breadcrumb names ${quoted(name)} at ${path}, which is no page in the model.`;
        },
        ambiguous: (name: string, candidates: readonly string[]) => {
            if (name === "" || candidates.length === 0) {
                return "A crumb of the breadcrumb matches several pages.";
            }
            return `The breadcrumb names ${quoted(name)}, which ${candidates.length} pages carry: ${candidates.join(", ")}.`;
        },
        twin: (name: string, page: string, terms: readonly string[]) => {
            if (name === "" || page === "") {
                return "A crumb's name is shared by a page and a store category, so the page was taken.";
            }
            const over = terms.length === 0 ? "a store category of the same name" : `the store category ${terms.join(", ")}`;
            return `The breadcrumb names ${quoted(name)}: the page ${page} was taken over ${over}.`;
        },
        cycle: (paths: readonly string[]) =>
            paths.length === 0
                ? "The parents loop, so the loop is broken here."
                : `The parents loop through ${paths.join(" › ")}, so the loop is broken here.`,
        differs: (sheet: readonly string[], site: readonly string[]) => {
            if (sheet.length === 0) {
                return "The workbook places this page under other parents than the site shows.";
            }
            const shown = site.length === 0 ? "at the top" : `under ${site.join(" › ")}`;
            return `The workbook puts this page under ${sheet.join(" › ")}; the site shows it ${shown}.`;
        },
        rowAmbiguous: (sheet: string, candidates: readonly string[]) => {
            const from = sheet === "" ? "the workbook" : `the sheet ${sheet}`;
            return candidates.length === 0
                ? `A row of ${from} matches several pages.`
                : `A row of ${from} matches ${candidates.length} pages: ${candidates.join(", ")}.`;
        },
        rowWithoutPage: (sheet: string, planned: string, product: boolean) => {
            const from = sheet === "" ? "the workbook" : `the sheet ${sheet}`;
            if (product) {
                return `A product row of ${from} waits for its product in the store.`;
            }
            return planned === ""
                ? `A row of ${from} has no page on the site yet.`
                : `A row of ${from} has no page on the site yet; it is planned at ${planned}.`;
        },
        outdated: (version: string, required: string) =>
            required === ""
                ? "The companion plugin is out of date."
                : `The companion plugin is version ${version === "" ? "unknown" : version} and ${required} is needed.`,
        field: (field: string) => `The ${field} was edited on the site since it was last read or written.`,
        body: "The body was edited on the site since it was last read or written.",
        unreachable: (status: number) =>
            status > 0
                ? `The site answered ${status} when the page was fetched.`
                : "The page could not be fetched from the site.",
        redirected: (target: string) =>
            target === "" ? "Visiting the page landed on another address." : `Visiting the page landed on ${target}.`,
        builder: (name: string) => `Made with ${name}, so its body is read only.`,
        technical: (reason: string) => `Read as a technical page: ${reason}.`,
        titleDiffers: (planned: string, site: string) => `The plan says ${quoted(planned)}, the site says ${quoted(site)}.`,
        urlForms: (absolute: number, relative: number) =>
            `The site's pages mix link forms: ${absolute} absolute, ${relative} relative.`,
        cardProposed: (name: string) =>
            name === ""
                ? "The agent proposed a card for an entity; accept or dismiss it on the Graph screen."
                : `The agent proposed a card for ${name}; accept or dismiss it on the Graph screen.`,
        generic: "The observation flagged something here.",
    },
    fieldNames: {
        title: "title",
        h1: "H1",
        excerpt: "excerpt",
        body: "body",
        seo_title: "SEO title",
        seo_description: "SEO description",
        seo_canonical: "canonical address",
        product_description: "product description",
        product_short_description: "short description",
        variation_description: "variation description",
        featured_image: "featured image",
        image_alt: "image alt text",
        image_title: "image title",
        upsells: "upsells",
        cross_sells: "cross-sells",
        meta: "custom field",
    } as Readonly<Record<string, string>>,
    unknownField: "field",
    wpTypes: {
        page: "Page",
        post: "Post",
        product: "Product",
        product_cat: "Product category",
    },
    wpStatuses: {
        publish: "Published",
        draft: "Draft",
        pending: "Pending review",
        private: "Private",
        future: "Scheduled",
        trash: "In the trash",
        inherit: "Inherited",
    } as Readonly<Record<string, string>>,
    taxonomies: {
        category: "Category",
        post_tag: "Tag",
        product_cat: "Product category",
        product_tag: "Product tag",
    } as Readonly<Record<string, string>>,
    linkTypes: {
        breadcrumb: "Breadcrumb",
        listing: "Listing",
        navigation: "Navigation",
        contextual: "Contextual",
        unknown: "Unclassified",
    },
    linkSources: {
        raw: "in the body",
        render: "in the render",
    },
    regionKinds: {
        breadcrumb: "Breadcrumb",
        listing: "Listing",
        navigation: "Navigation",
        content: "Content",
        content_dynamic: "Dynamic content",
        generated: "Generated",
        unknown: "Unclassified",
    },
    parentReasons: {
        jsonld_breadcrumb: "JSON-LD breadcrumb",
        aria_breadcrumb: "Breadcrumb navigation",
        wp_parent: "WordPress parent",
        url: "Address",
        workbook: "Workbook",
        none: "No parent",
    },
    parentSources: {
        jsonld_breadcrumb: "from the JSON-LD breadcrumb",
        aria_breadcrumb: "from the breadcrumb navigation",
        wp_parent: "from the WordPress parent",
        url: "from the address",
        workbook: "from the workbook",
        none: "with no source",
    },
    parentMethods: {
        href: "by its link",
        redirect: "through a redirect",
        name: "by its name",
        "": "without a breadcrumb",
    },
    parentMethodNames: {
        href: "By link",
        redirect: "Through a redirect",
        name: "By name",
        "": "Without a breadcrumb",
    },
    trailSources: {
        jsonld: "JSON-LD breadcrumb list",
        nav: "Breadcrumb navigation",
    },
    technicalReasons: {
        posts_page: "Posts page",
        privacy: "Privacy policy",
        shop: "Shop page",
        cart: "Cart",
        checkout: "Checkout",
        account: "Account page",
        terms: "Terms and conditions",
        shortcode: "Store shortcode",
        noindex: "Hidden from search engines",
        workbook: "Marked in the workbook",
        manual: "Marked by hand",
        "": "Not technical",
    },
    rootStates: {
        found: "Content found",
        empty_raw: "Empty body",
        dynamic_only: "Dynamic content only",
        not_found: "Content not located",
        archive: "Archive page",
    },
    findingCodes: {
        parent_unresolved: "Crumb without a page",
        parent_ambiguous: "Crumb matches several pages",
        parent_cycle: "Parents loop",
        parent_name_twin: "Page and store category share a name",
        parent_differs: "Workbook parent differs",
        plugin_outdated: "Plugin out of date",
        field_changed_on_site: "Field edited on the site",
        body_changed_on_site: "Body edited on the site",
        render_unreachable: "Page could not be fetched",
        render_redirected: "Visit redirected",
        builder_detected: "Page builder",
        technical_detected: "Technical page",
        page_title_differs: "Title differs from the plan",
        page_h1_differs: "H1 differs from the plan",
        url_form_mixed: "Mixed link forms",
        entity_card_proposed: "Card proposed",
        row_ambiguous: "Workbook row matches several pages",
        row_without_page: "Workbook row without a page",
    },
    severities: {
        info: "Note",
        warning: "Warning",
        error: "Error",
    },
    urlForms: {
        auto: "Automatic",
        absolute: "Absolute",
        relative: "Relative",
    },
    blockKinds: {
        paragraph: "Paragraph",
        heading: "Heading",
        list: "List",
        quote: "Quote",
        html: "HTML",
        shortcode: "Shortcode",
        wp: "Block",
        foreign: "Other block",
    },
    bodyKinds: {
        classic: "Classic editor",
        blocks: "Block editor",
    },
};
