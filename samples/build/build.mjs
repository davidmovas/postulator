import { mkdir, writeFile } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import ExcelJS from "exceljs";
import { planHeaders, planRows, messyHeaders, messyRows } from "./plan.mjs";
import { crawlHeaders, crawlRows, feedHeaders, feedRows } from "./site.mjs";
import {
  bikesRows,
  brokenHeaders,
  brokenRows,
  componentsRows,
  crawlHeaders as messyCrawlHeaders,
  crawlRows as messyCrawlRows,
  guidesRows,
  indentHeaders,
  indentRows,
  notesHeaders,
  notesRows,
  plainHeaders,
} from "./messy.mjs";

const here = dirname(fileURLToPath(import.meta.url));
const out = resolve(here, "..");

async function workbook(name, headers, rows, file) {
  const book = new ExcelJS.Workbook();
  book.creator = "Postulator samples";
  book.created = new Date(Date.UTC(2026, 8, 21));
  book.modified = book.created;
  const sheet = book.addWorksheet(name);
  sheet.addRow(headers);
  for (const row of rows) {
    sheet.addRow(row);
  }
  sheet.getRow(1).font = { bold: true };
  sheet.columns.forEach((column, index) => {
    const widest = [headers[index], ...rows.map((row) => row[index])]
      .map((value) => String(value ?? "").length)
      .reduce((a, b) => Math.max(a, b), 8);
    column.width = Math.min(widest + 2, 54);
  });
  await book.xlsx.writeFile(join(out, file));
  return rows.length;
}

async function sheets(parts, file) {
  const book = new ExcelJS.Workbook();
  book.creator = "Postulator samples";
  book.created = new Date(Date.UTC(2026, 8, 22));
  book.modified = book.created;

  let total = 0;
  for (const part of parts) {
    const sheet = book.addWorksheet(part.name);
    if (part.headers.length > 0) {
      sheet.addRow(part.headers);
      sheet.getRow(1).font = { bold: true };
    }
    for (const row of part.rows) {
      sheet.addRow(row);
    }
    sheet.columns.forEach((column) => {
      column.width = 28;
    });
    total += part.rows.length;
  }

  await book.xlsx.writeFile(join(out, file));
  return total;
}

function field(value) {
  const text = String(value ?? "");
  if (text.includes(",") || text.includes('"') || text.includes("\n")) {
    return '"' + text.replaceAll('"', '""') + '"';
  }
  return text;
}

async function separated(headers, rows, file) {
  const lines = [headers, ...rows].map((row) => row.map(field).join(","));
  await writeFile(join(out, file), lines.join("\r\n") + "\r\n", "utf8");
  return rows.length;
}

async function main() {
  await mkdir(out, { recursive: true });
  const written = [
    ["entity-plan.xlsx", await workbook("Topical plan", planHeaders, planRows, "entity-plan.xlsx")],
    ["site-export.xlsx", await workbook("internal_html", crawlHeaders, crawlRows, "site-export.xlsx")],
    ["product-feed.csv", await separated(feedHeaders, feedRows, "product-feed.csv")],
    ["messy-plan.xlsx", await workbook("Topical plan", messyHeaders, messyRows, "messy-plan.xlsx")],
    [
      "messy-sheets.xlsx",
      await sheets(
        [
          { name: "Site map", headers: indentHeaders, rows: indentRows },
          { name: "Bikes", headers: plainHeaders, rows: bikesRows },
          { name: "Components", headers: plainHeaders, rows: componentsRows },
          { name: "Guides", headers: plainHeaders, rows: guidesRows },
          { name: "shop crawl (old)", headers: messyCrawlHeaders, rows: messyCrawlRows },
          { name: "Broken", headers: brokenHeaders, rows: brokenRows },
          { name: "Notes", headers: notesHeaders, rows: notesRows },
        ],
        "messy-sheets.xlsx",
      ),
    ],
  ];
  for (const [name, count] of written) {
    process.stdout.write(name + " " + count + " rows\n");
  }
}

await main();
