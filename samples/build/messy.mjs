const sections = [
  {
    slug: "electric-bikes",
    title: "Electric Bikes",
    keyword: "electric bikes",
    groups: [
      { slug: "commuter", title: "Commuter E-Bikes", keyword: "commuter electric bike" },
      { slug: "cargo", title: "Cargo E-Bikes", keyword: "cargo electric bike" },
      { slug: "folding", title: "Folding E-Bikes", keyword: "folding electric bike" },
      { slug: "mountain", title: "Electric Mountain Bikes", keyword: "electric mountain bike" },
      { slug: "gravel", title: "Electric Gravel Bikes", keyword: "electric gravel bike" },
    ],
  },
  {
    slug: "components",
    title: "E-Bike Components",
    keyword: "e-bike components",
    groups: [
      { slug: "batteries", title: "E-Bike Batteries", keyword: "e-bike battery" },
      { slug: "motors", title: "E-Bike Motors", keyword: "e-bike motor" },
      { slug: "chargers", title: "E-Bike Chargers", keyword: "e-bike charger" },
      { slug: "displays", title: "E-Bike Displays", keyword: "e-bike display" },
      { slug: "brakes", title: "E-Bike Brakes", keyword: "e-bike brakes" },
    ],
  },
  {
    slug: "guides",
    title: "Buying Guides",
    keyword: "e-bike buying guide",
    groups: [
      { slug: "beginners", title: "Beginner Guides", keyword: "e-bike guide for beginners" },
      { slug: "comparisons", title: "E-Bike Comparisons", keyword: "e-bike comparison" },
      { slug: "budget", title: "Budget Guides", keyword: "cheap electric bike" },
      { slug: "commuting", title: "Commuting Guides", keyword: "e-bike commuting" },
      { slug: "touring", title: "Touring Guides", keyword: "e-bike touring" },
    ],
  },
];

const makes = [
  "Volt", "Hauler", "Stowaway", "Summit", "Ridgeline", "PowerCell", "TorqueOne", "HubGlide",
  "RapidVolt", "ClearView", "Northwind", "Cobbler", "Drayton", "Kestrel", "Marlow", "Pennine",
  "Quarry", "Rushmere", "Sableton", "Thornbury",
];

const models = ["100", "220", "350", "500", "720", "900", "Air", "Lite", "Max", "Pro", "Sport", "Tour"];

function slugify(text) {
  return String(text)
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

function* products(limit) {
  let made = 0;
  for (const section of sections) {
    for (const group of section.groups) {
      for (const make of makes) {
        for (const model of models) {
          if (made >= limit) {
            return;
          }
          made += 1;
          const name = `${make} ${group.title.split(" ")[0]} ${model}`;
          yield {
            section,
            group,
            name,
            slug: slugify(`${make}-${model}`),
            path: `/${section.slug}/${group.slug}/${slugify(`${make}-${model}`)}/`,
            keyword: `${make.toLowerCase()} ${model.toLowerCase()}`,
            secondary: [`${make.toLowerCase()} ${model.toLowerCase()} review`, group.keyword, section.keyword],
          };
        }
      }
    }
  }
}

export const plainHeaders = ["Page path", "Page title", "Parent", "Primary keyword", "Secondary keywords", "Kind"];

function plainRows(limit, only) {
  const rows = [];
  for (const section of sections) {
    if (only !== undefined && section.slug !== only) {
      continue;
    }
    rows.push([`/${section.slug}/`, section.title, "", section.keyword, "", "hub"]);
    for (const group of section.groups) {
      rows.push([
        `/${section.slug}/${group.slug}/`, group.title, section.title, group.keyword,
        `${section.keyword}, ${group.keyword} uk`, "category",
      ]);
    }
  }
  for (const item of products(limit)) {
    if (only !== undefined && item.section.slug !== only) {
      continue;
    }
    rows.push([
      item.path, item.name, item.group.title, item.keyword, item.secondary.join(", "), "product",
    ]);
  }
  return rows;
}

export const bikesRows = plainRows(3000, "electric-bikes");
export const componentsRows = plainRows(3000, "components");
export const guidesRows = plainRows(3000, "guides");

export const indentHeaders = [];

export const indentRows = (() => {
  const rows = [[], [], ["   "], ["", "Prepared by the agency, do not edit"], []];
  rows.push(["", "somedomain.uk"]);
  for (const section of sections) {
    rows.push(["", "", `/${section.slug}`]);
    for (const group of section.groups) {
      rows.push(["", "", "", group.slug]);
      let made = 0;
      for (const item of products(3000)) {
        if (item.group.slug !== group.slug || item.section.slug !== section.slug || made >= 12) {
          continue;
        }
        made += 1;
        rows.push(["", "", "", "", `${item.slug}/`]);
      }
    }
  }
  return rows;
})();

export const crawlHeaders = ["URL", "Name", "Focus keyword", "Tags", "Pillar", "Status", "Inlinks"];

export const crawlRows = (() => {
  const rows = [];
  let index = 0;
  for (const item of products(1000)) {
    index += 1;
    const host = index % 3 === 0 ? "HTTPS://SomeDomain.UK" : "https://somedomain.uk";
    const tail = index % 4 === 0 ? item.path.replace(/\/$/, "") : item.path;
    const address = index % 7 === 0 ? tail.toUpperCase() : `${host}${tail}`;
    rows.push([
      address, item.name, item.keyword, item.secondary.join(" | "), item.group.title,
      index % 5 === 0 ? "live" : "planned", String(200 - (index % 50)),
    ]);
    if (index % 37 === 0) {
      rows.push([`${host}${item.path}`, `${item.name} (duplicate)`, item.keyword, "", item.group.title, "planned", "1"]);
    }
  }
  return rows;
})();

export const brokenHeaders = plainHeaders;

export const brokenRows = [
  ["/laws/", "E-Bike Laws", "", "e-bike laws", "", "hub"],
  ["/laws/uk/", "E-Bike Law in the UK", "E-Bike Laws", "uk e-bike law", "", "topic"],
  ["/laws/germany/", "E-Bike Law in Germany", "Rules Nobody Declared", "germany e-bike law", "", "topic"],
  ["/laws/ france /", "A path with spaces", "E-Bike Laws", "france e-bike law", "", "topic"],
  ["", "A row that names nothing", "", "", "", ""],
  ["/laws/loop-a/", "Loop A", "Loop B", "loop a", "", "topic"],
  ["/laws/loop-b/", "Loop B", "Loop A", "loop b", "", "topic"],
  ["/laws/itself/", "Its own parent", "Its own parent", "itself", "", "topic"],
  ["/laws/uk/", "The same path again", "E-Bike Laws", "uk e-bike law", "", "topic"],
];

export const notesHeaders = ["Note"];

export const notesRows = [
  ["Sheet one is the site map the client keeps by hand."],
  ["The product sheets were exported from the old shop."],
  ["Nobody remembers who made the crawl sheet."],
];
