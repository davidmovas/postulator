import { mkdir, readFile } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";
import process from "node:process";

import { chromium } from "playwright-core";

const usage = `postulator ui screenshots

  node scripts/shot.mjs --port 9222 --route "#/sites" --out shots/sites.png
      [--width 1280] [--height 820] [--wait 800] [--site <id>]
      [--clip "<x>,<y>,<w>,<h>"]
      [--click "<css>"] [--type "<css>=<text>"] [--key "<Key>"] [--fits "<css>"]
      [--expect "<css>"]

  node scripts/shot.mjs --port 9222 --routes scripts/routes.json --out-dir shots
      [--from <name>] [--until <name>] [--dock]

A walk closes the agent dock and reloads once before its first shot, so the
set does not depend on what the last session left open. --from starts the walk
at the named shot and --until stops it after the named one, and --dock opens
the dock after that reload, for a walk started past the shots that open it.

The window is the one "task ui:run" started; it is reached over the DevTools
protocol and never launched here. --click, --type and --key are applied in the
order they are written on the command line, after the route has settled.

A route containing :siteId is resolved from --site when it is given. Otherwise
the script opens #/sites and reads the first [data-site-id] row of the table,
so the seeded site does not have to be known in advance. A routes.json entry
with "siteName" takes the row whose [data-site-name] is that name instead.

An entry with "timed": "<css>" reports how long the window took from the
route's navigation to the first visible match, and a step
{"timed": "<label>", "click": "<css>", "until": "<css>"} reports how long a
click took to show its first match. A step {"scroll": "<css>", "label"} scrolls
the named list through 60 frames and reports the average and the worst frame.
Neither is asserted: the numbers are printed as "timed" and "scroll" lines.

--clip writes the named rectangle of the viewport at 1:1 instead of the whole
window, which is how a strip is judged at pixel scale.

--fits fails the shot when the named element, or anything inside it that
scrolls, holds content wider than itself: the proof that a screen fits its
panel rather than scrolling sideways.

--expect waits up to 15 s for the named element and fails the shot when it
never appears; in routes.json an expect step may add "count", the exact number
of matches to wait for, and "timeout" in milliseconds.

A click step with "shift": true holds Shift down for that click, which is how a
range is ticked.

routes.json is an array of
  { "name", "route", "width", "height", "wait", "site", "siteName", "clip", "timed",
    "steps": [ {"click": "<css>"}, {"click": "<css>", "shift": true}, {"type": "<css>=<text>"},
               {"key": "Control+K"}, {"wait": 500}, {"fits": "<css>"},
               {"expect": "<css>", "count": 2, "timeout": 15000},
               {"timed": "<label>", "click": "<css>", "until": "<css>"},
               {"scroll": "<css>", "label": "<label>"} ] }
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
    "--fits",
    "--expect",
    "--from",
    "--until",
]);

function parse(argv) {
    const options = { port: 9222, width: 1280, height: 820, wait: 800, steps: [] };
    for (let index = 0; index < argv.length; index += 1) {
        const flag = argv[index];
        if (flag === "--help" || flag === "-h") {
            return { help: true, steps: [] };
        }
        if (flag === "--dock") {
            options.dock = true;
            continue;
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
            case "--fits":
                options.steps.push({ fits: value });
                break;
            case "--expect":
                options.steps.push({ expect: value });
                break;
            case "--from":
                options.from = value;
                break;
            case "--until":
                options.until = value;
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

async function resolveNamed(page, name, wait) {
    await go(page, "#/sites", wait);
    const row = page.locator(`[data-site-id][data-site-name=${JSON.stringify(name)}]`).first();
    await row.waitFor({ state: "attached", timeout: 15000 }).catch(() => {
        throw new Error(`no site named ${name} is listed`);
    });
    const id = await row.getAttribute("data-site-id");
    if (id === null || id === "") {
        throw new Error(`the site named ${name} has no id`);
    }
    return id;
}

async function timedUntil(page, css, started, label) {
    await page
        .locator(css)
        .first()
        .waitFor({ state: "visible", timeout: 120000 })
        .catch(() => {
            throw new Error(`nothing matched ${css} within 120 s for ${label}`);
        });
    process.stdout.write(`timed ${label}: ${Date.now() - started} ms\n`);
}

async function scrolled(page, css, label) {
    const measured = await page.evaluate(async (selector) => {
        const list = document.querySelector(selector);
        if (list === null) {
            return null;
        }
        const gaps = [];
        let last = performance.now();
        for (let frame = 0; frame < 60; frame += 1) {
            list.scrollTop += 240;
            await new Promise((done) => {
                requestAnimationFrame(() => {
                    done(undefined);
                });
            });
            const now = performance.now();
            gaps.push(now - last);
            last = now;
        }
        const total = gaps.reduce((sum, gap) => sum + gap, 0);
        return { frames: gaps.length, average: total / gaps.length, worst: Math.max(...gaps), reached: list.scrollTop };
    }, css);
    if (measured === null) {
        throw new Error(`nothing matched ${css} to scroll for ${label}`);
    }
    process.stdout.write(
        `scroll ${label}: ${measured.frames} frames, ${measured.average.toFixed(1)} ms on average, ${measured.worst.toFixed(1)} ms at worst, ${Math.round(measured.reached)} px down\n`,
    );
}

async function fits(page, css) {
    const sideways = await page.evaluate((selector) => {
        const root = document.querySelector(selector);
        if (root === null) {
            return `nothing matches ${selector}`;
        }
        if (root.getBoundingClientRect().right > document.documentElement.clientWidth + 1) {
            return `${selector} reaches past the window's right edge`;
        }
        for (const held of [root, ...root.querySelectorAll("*")]) {
            const overflow = window.getComputedStyle(held).overflowX;
            const scrolls = overflow === "auto" || overflow === "scroll";
            if (scrolls && held.clientWidth > 0 && held.scrollWidth > held.clientWidth + 1) {
                return `a ${held.tagName.toLowerCase()} inside ${selector} scrolls sideways: ${held.scrollWidth}px of content in ${held.clientWidth}px`;
            }
        }
        return null;
    }, css);
    if (sideways !== null) {
        throw new Error(sideways);
    }
}

async function expectIn(page, step) {
    const timeout = step.timeout ?? 15000;
    const matches = page.locator(step.expect);
    if (step.count === undefined) {
        await matches
            .first()
            .waitFor({ state: "attached", timeout })
            .catch(() => {
                throw new Error(`nothing matched ${step.expect} within ${timeout} ms`);
            });
        return;
    }
    const deadline = Date.now() + timeout;
    let found = await matches.count();
    while (found !== step.count && Date.now() < deadline) {
        await page.waitForTimeout(200);
        found = await matches.count();
    }
    if (found !== step.count) {
        throw new Error(`${found} elements matched ${step.expect} after ${timeout} ms, want ${step.count}`);
    }
}

async function apply(page, steps) {
    for (const step of steps) {
        if (step.timed !== undefined) {
            if (step.click === undefined || step.until === undefined) {
                throw new Error(`a timed step needs a click and an until: ${JSON.stringify(step)}`);
            }
            await page.bringToFront();
            const started = Date.now();
            await page.click(step.click);
            await timedUntil(page, step.until, started, step.timed);
            continue;
        }
        if (step.scroll !== undefined) {
            await scrolled(page, step.scroll, step.label ?? step.scroll);
            continue;
        }
        if (step.click !== undefined) {
            await page.bringToFront();
            await page.click(step.click, step.shift === true ? { modifiers: ["Shift"] } : {});
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
        if (step.fits !== undefined) {
            await fits(page, step.fits);
            continue;
        }
        if (step.expect !== undefined) {
            await expectIn(page, step);
            continue;
        }
        throw new Error(`a step names none of timed, scroll, click, type, key, wait, fits or expect: ${JSON.stringify(step)}`);
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

async function shoot(page, shot, resolveSiteId, resolveNamedId) {
    await page.keyboard.press("Escape");
    await page.waitForTimeout(200);
    const siteId = () =>
        shot.site ?? (shot.siteName === undefined ? resolveSiteId(shot.wait) : resolveNamedId(shot.siteName, shot.wait));
    const route = shot.route.includes(":siteId") ? shot.route.replaceAll(":siteId", await siteId()) : shot.route;
    await page.setViewportSize({ width: shot.width, height: shot.height });
    if (shot.timed !== undefined) {
        const started = Date.now();
        await page.evaluate((hash) => {
            window.location.hash = hash;
        }, hashOf(route));
        await timedUntil(page, shot.timed, started, shot.name);
    }
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
                name: options.out,
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
    const read = JSON.parse(await readFile(resolve(options.routes), "utf8"));
    if (!Array.isArray(read)) {
        throw new Error(`${options.routes} must hold an array of routes`);
    }
    const first = options.from === undefined ? 0 : read.findIndex((entry) => entry.name === options.from);
    if (first < 0) {
        throw new Error(`no route in ${options.routes} is named ${options.from}`);
    }
    const last = options.until === undefined ? read.length - 1 : read.findIndex((entry) => entry.name === options.until);
    if (last < 0) {
        throw new Error(`no route in ${options.routes} is named ${options.until}`);
    }
    const listed = read.slice(first, last + 1);
    return listed.map((entry) => {
        if (typeof entry.name !== "string" || typeof entry.route !== "string") {
            throw new Error(`a route entry needs a name and a route: ${JSON.stringify(entry)}`);
        }
        return {
            name: entry.name,
            route: entry.route,
            out: join(options.outDir, `${entry.name}.png`),
            width: entry.width ?? options.width,
            height: entry.height ?? options.height,
            wait: entry.wait ?? options.wait,
            site: entry.site ?? options.site,
            siteName: entry.siteName,
            timed: entry.timed,
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
    const named = new Map();
    const resolverByName = async (name, wait) => {
        if (!named.has(name)) {
            named.set(name, await resolveNamed(page, name, wait));
        }
        return named.get(name);
    };
    try {
        if (options.routes !== undefined) {
            await rest(page);
            if (options.dock === true) {
                await go(page, "#/sites", 800);
                const expand = page.locator('[aria-label="Expand the agent dock"]');
                if ((await expand.count()) > 0) {
                    await page.bringToFront();
                    await expand.first().click();
                }
                await expectIn(page, { expect: "[data-agent-dock]" });
                await page.waitForTimeout(900);
            }
        }
        for (const shot of shots) {
            await shoot(page, shot, resolver, resolverByName);
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
