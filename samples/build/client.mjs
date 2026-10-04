export const groupsHeaders = ["Root Entity", "Category", "Subcategory", "Recommended URL Layer", "Title", "H1", "Keywords"];

export const groupsRows = [
  ["Peptides", "", "", "/peptides/", "Research peptides", "Peptides", "research peptides (22000), buy peptides (9,900)"],
  ["Peptides", "BPC-157", "", "/peptides/bpc-157/", "BPC-157 peptide", "BPC-157", "bpc 157 (12000), buy bpc 157 (5,400), bpc-157"],
  ["Peptides", "BPC-157", "Liquid", "/peptides/bpc-157/liquid/", "BPC-157 liquid", "BPC-157 Liquid", "bpc 157 liquid (900), bpc 157 injection"],
  ["Peptides", "BPC-157", "Powder", "/peptides/bpc-157/powder/", "BPC-157 powder", "BPC-157 Powder", ""],
  ["Peptides", "TB-500", "", "/peptides/tb-500/", "TB-500 peptide", "TB-500", "tb 500 (8100), tb500 peptide"],
  ["Peptides", "TB-500", "Liquid", "/peptides/tb-500/liquid/", "TB-500 liquid", "TB-500 Liquid", "tb 500 liquid (480)"],
  ["Peptides", "TB-500", "Capsules", "/peptides/tb-500/capsules/", "TB-500 capsules", "TB-500 Capsules", "tb 500 capsules [1.2k]"],
  ["Peptides", "—", "", "/peptides/storage/", "How to store peptides", "Storing peptides", "how to store peptides (320); peptide storage"],
];

export const catalogHeaders = ["Category", "Subcategory", "URL", "Title", "H1", "Keywords"];

export const catalogRows = [
  ["Peptides", "BPC-157", "/shop/bpc-157-5mg/", "BPC-157 5 mg", "BPC-157 5 mg vial", "bpc 157 5mg (1,300)"],
  ["Peptides", "BPC-157", "/shop/bpc-157-10mg/", "BPC-157 10 mg", "BPC-157 10 mg vial", "bpc 157 10mg (880)"],
  ["Peptides", "TB-500", "/shop/tb-500-5mg/", "TB-500 5 mg", "TB-500 5 mg vial", "tb 500 5mg"],
  ["Blends", "Recovery", "/shop/recovery-blend/", "Recovery blend", "BPC-157 and TB-500 blend", "bpc 157 tb 500 blend (720)"],
];

export const wideHeaders = [
  "Entity ID", "Primary Entity", "Entity Level", "Entity Type", "Entity Name", "Canonical URL", "Parent Entity",
  "Category", "Subcategory", "Title", "H1", "Intent Owner", "Page Template", "Notes",
];

export const wideRows = [
  ["E-001", "YES", "Category", "Commercial Taxonomy", "Peptides", "/peptides/", "", "Peptides", "", "Research peptides", "Peptides", "Commercial", "", ""],
  [
    "E-014", "NO", "Compound/Product", "Product Owner", "BPC-157", "/peptides/bpc-157/", "Peptides", "Peptides", "BPC-157",
    "BPC-157 peptide", "BPC-157", "Commercial Product", "", "Sold as a 10 ml vial",
  ],
  [
    "E-030", "NO", "GEO", "Geographic Entity", "Peptides in Canada", "/peptides/canada/", "Peptides", "Peptides", "",
    "Peptides in Canada", "Buying peptides in Canada", "GEO Commercial", "", "Shipping rules differ by province",
  ],
];

export const variationsHeaders = ["Parent Product Entity", "URL", "Detected Form / Variation", "Entity?", "Reason"];

export const variationsRows = [
  ["BPC-157", "/peptides/bpc-157/capsules/", "Capsules", "NO", "The shop sells it in capsules too"],
  ["TB-500", "/peptides/tb-500/liquid/", "Liquid", "NO", "Already planned in the group sheet"],
  ["TB-500", "/peptides/tb-500/nasal-spray/", "Nasal spray", "NO", "A form the shop added"],
];
