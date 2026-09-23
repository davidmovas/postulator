import { mkdir, readFile } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";
import process from "node:process";

import { chromium } from "playwright-core";

const usage = `postulator ui screenshots

  node scripts/shot.mjs --port 9222 --route "#/sites" --out shots/sites.png
      [--width 1280] [--height 820] [--wait 800] [--site <id>]
      [--clip "<x>,<y>,<w>,<h>"]
      [--click "<css>"] [--type "<css>=<text>"] [--key "<Key>"]

  node scripts/shot.mjs --port 9222 --routes scripts/routes.json --out-dir shots

A walk closes the agent dock and reloads once before its first shot, so the
set does not depend on what the last session left open.

The window is the one "task ui:run" started; it is reached over the DevTools
protocol and never launched here. --click, --type and --key are applied in the
order they are written on the command line, after the route has settled.

A route containing :siteId is resolved from --site when it is given. Otherwise
the script opens #/sites and reads the first [data-site-id] row of the table,
so the seeded site does not have to be known in advance.

--clip writes the named rectangle of the viewport at 1:1 instead of the whole
window, which is how a strip is judged at pixel scale.

routes.json is an array of
  { "name", "route", "width", "height", "wait", "site", "clip",
    "steps": [ {"click": "<css>"}, {"type": "<css>=<text>"},
               {"key": "Control+K"}, {"wait": 500} ] }
and each name becomes <out-dir>/<name>.png.`;

const flags = new Set([
    "--port",
    "--route",
    "--out",
    "--width",
    "--height",
    "--wait",
    "--site",
    "--clip",
    "--routes",
    "--out-dir",
    "--click",
    "--type",
    "--key",
]);

function parse(argv) {
    const options = { port: 9222, width: 1280, height: 820, wait: 800, steps: [] };
    for (let index = 0; index < argv.length; index += 1) {
        const flag = argv[index];
        if (flag === "--help" || flag === "-h") {
            return { help: true, steps: [] };
        }
        if (!flags.has(flag)) {
            throw new Error(`unknown flag ${flag}`);
        }
        const value = argv[index + 1];
        if (value === undefined) {
            throw new Error(`${flag} needs a value`);
        }
        index += 1;
        switch (flag) {
            case "--port":
                options.port = Number(value);
                break;
            case "--width":
                options.width = Number(value);
                break;
            case "--height":
                options.height = Number(value);
                break;
            case "--wait":
                options.wait = Number(value);
                break;
            case "--route":
                options.route = value;
                break;
            case "--out":
                options.out = value;
                break;
            case "--site":
                options.site = value;
                break;
            case "--clip":
                options.clip = value;
                break;
            case "--routes":
                options.routes = value;
                break;
            case "--out-dir":
                options.outDir = value;
                break;
            case "--click":
                options.steps.push({ click: value });
                break;
            case "--type":
                options.steps.push({ type: value });
                break;
            case "--key":
                options.steps.push({ key: value });
                break;
            default:
                throw new Error(`unhandled flag ${flag}`);
        }
    }
    return options;
}

const internalScheme = /^(about|chrome|edge|devtools|chrome-error|chrome-extension):/;

async function attach(port) {
    const browser = await chromium.connectOverCDP(`http://127.0.0.1:${port}`);
    const context = browser.contexts()[0];
    if (context === undefined) {
        await browser.close();
        throw new Error(`nothing is listening for pages on port ${port}; start the window with "task ui:run"`);
    }
    const pages = context.pages();
    const page = pages.find((held) => !internalScheme.test(held.url())) ?? pages[0];
    if (page === undefined) {
        await browser.close();
        throw new Error(`the window on port ${port} has no page`);
    }
    return { browser, page };
}

async function settle(page, wait) {
    await page.waitForLoadState("networkidle").catch(() => undefined);
    await page.waitForTimeout(wait);
}

function clipOf(value) {
    if (value === undefined) {
        return undefined;
    }
    const parts = value.split(",").map((part) => Number(part.trim()));
    if (parts.length !== 4 || parts.some((part) => !Number.isFinite(part))) {
        throw new Error(`--clip takes "<x>,<y>,<w>,<h>", got ${value}`);
    }
    return { x: parts[0], y: parts[1], width: parts[2], height: parts[3] };
}

function hashOf(route) {
    return route.startsWith("#") ? route.slice(1) : route;
}

async function go(page, route, wait) {
    const wanted = hashOf(route);
    for (let attempt = 0; attempt < 3; attempt += 1) {
        await page.evaluate((hash) => {
            window.location.hash = hash;
        }, wanted);
        await settle(page, wait);
        const landed = await page.evaluate(() => window.location.hash.slice(1));
        if (landed === wanted) {
            return;
        }
    }
    throw new Error(`the window would not stay on ${wanted}`);
}

async function resolveSite(page, wait) {
    await go(page, "#/sites", wait);
    const id = await page.getAttribute("[data-site-id]", "data-site-id");
    if (id === null || id === "") {
        throw new Error("no site is listed, so :siteId cannot be resolved; pass --site <id>");
    }
    return id;
}

async function apply(page, steps) {
    for (const step of steps) {
        if (step.click !== undefined) {
            await page.bringToFront();
            await page.click(step.click);
            continue;
        }
        if (step.type !== undefined) {
            const split = step.type.indexOf("=");
            if (split < 0) {
                throw new Error(`--type takes "<css>=<text>", got ${step.type}`);
            }
            await page.fill(step.type.slice(0, split), step.type.slice(split + 1));
            continue;
        }
        if (step.key !== undefined) {
            await page.keyboard.press(step.key);
            continue;
        }
        if (step.wait !== undefined) {
            await page.waitForTimeout(step.wait);
            continue;
        }
        throw new Error(`a step names none of click, type, key or wait: ${JSON.stringify(step)}`);
    }
}

async function rest(page) {
    await page.evaluate(() => {
        for (const key of ["postulator.dock.open", "postulator.dock.width"]) {
            try {
                window.localStorage.removeItem(key);
            } catch {
                return;
            }
        }
    });
    await page.reload({ waitUntil: "networkidle" });
    await page.waitForTimeout(1200);
}

async function shoot(page, shot, resolveSiteId) {
    await page.keyboard.press("Escape");
    await page.waitForTimeout(200);
    const route = shot.route.includes(":siteId")
        ? shot.route.replaceAll(":siteId", shot.site ?? (await resolveSiteId(shot.wait)))
        : shot.route;
    await page.setViewportSize({ width: shot.width, height: shot.height });
    await go(page, route, shot.wait);
    await apply(page, shot.steps);
    await mkdir(dirname(shot.out), { recursive: true });
    await page.bringToFront();
    const clip = clipOf(shot.clip);
    await page.screenshot(clip === undefined ? { path: shot.out } : { path: shot.out, clip });
    process.stdout.write(`${shot.out}\n`);
}

async function planned(options) {
    if (options.routes === undefined) {
        if (options.route === undefined || options.out === undefined) {
            throw new Error("--route and --out are required without --routes; see --help");
        }
        return [
            {
                route: options.route,
                out: options.out,
                width: options.width,
                height: options.height,
                wait: options.wait,
                site: options.site,
                clip: options.clip,
                steps: options.steps,
            },
        ];
    }
    if (options.outDir === undefined) {
        throw new Error("--routes needs --out-dir");
    }
    const listed = JSON.parse(await readFile(resolve(options.routes), "utf8"));
    if (!Array.isArray(listed)) {
        throw new Error(`${options.routes} must hold an array of routes`);
    }
    return listed.map((entry) => {
        if (typeof entry.name !== "string" || typeof entry.route !== "string") {
            throw new Error(`a route entry needs a name and a route: ${JSON.stringify(entry)}`);
        }
        return {
            route: entry.route,
            out: join(options.outDir, `${entry.name}.png`),
            width: entry.width ?? options.width,
            height: entry.height ?? options.height,
            wait: entry.wait ?? options.wait,
            site: entry.site ?? options.site,
            clip: entry.clip ?? options.clip,
            steps: entry.steps ?? [],
        };
    });
}

async function main() {
    const options = parse(process.argv.slice(2));
    if (options.help === true) {
        process.stdout.write(`${usage}\n`);
        return;
    }
    const shots = await planned(options);
    const { browser, page } = await attach(options.port);
    let seeded = options.site;
    const resolver = async (wait) => {
        seeded ??= await resolveSite(page, wait);
        return seeded;
    };
    try {
        if (options.routes !== undefined) {
            await rest(page);
        }
        for (const shot of shots) {
            await shoot(page, shot, resolver);
        }
    } finally {
        await browser.close();
    }
}

try {
    await main();
} catch (thrown) {
    process.stderr.write(`${thrown instanceof Error ? thrown.message : String(thrown)}\n`);
    process.exitCode = 1;
}
