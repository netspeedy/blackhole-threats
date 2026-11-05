// Drive the site in headless Chrome: the theme controls, the header and its tabs, the version
// menu, and an archived line.
//
// It builds what it tests, so it needs no release metadata: the site once as an archived line
// (v9.8, last released as v9.8.7) and once as the latest (v9.9.0), with the archived one unpacked
// under <root>/v9.8/ the way the Pages workflow does it. One page is left out of the archive, to
// check that switching to a line without the page lands on the line's first page. The site root,
// the tabs and the pages it visits all come from the build, so the same script works on any
// project's site. The builds go to website/.test/.
//
// Run it with `make website-test`, which installs the Chrome build Puppeteer pins first. It prints
// one line per check and exits non-zero if any failed.
import { spawnSync } from "node:child_process";
import { cpSync, createReadStream, existsSync, mkdirSync, readFileSync, readdirSync, rmSync, statSync, writeFileSync } from "node:fs";
import { createServer } from "node:http";
import path from "node:path";
import { fileURLToPath } from "node:url";

import puppeteer from "puppeteer";

const website = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const work = path.join(website, ".test");
const ARCHIVED = { version: "v9.8", release: "v9.8.7" };
const LATEST = { version: "v9.9", release: "v9.9.0" };

// ------------------------------------------------------------------ fixtures

function build(name, env) {
  const outDir = path.join(work, name);
  const result = spawnSync(path.join(website, "node_modules", ".bin", "astro"), ["build", "--force", "--outDir", outDir], {
    cwd: website,
    env: { ...process.env, ...env },
    encoding: "utf8",
  });
  if (result.status !== 0) {
    process.stderr.write(result.stdout + result.stderr);
    throw new Error(`the ${name} build failed`);
  }
  return outDir;
}

const release = (tag) => ({ tag_name: tag, published_at: "2026-01-01T00:00:00Z", html_url: `https://example.invalid/releases/${tag}` });

rmSync(work, { recursive: true, force: true });
mkdirSync(work, { recursive: true });
writeFileSync(path.join(work, "archived.json"), JSON.stringify({ latest_release: release(ARCHIVED.release) }));
writeFileSync(
  path.join(work, "latest.json"),
  JSON.stringify({ latest_release: release(LATEST.release), docs_versions: [ARCHIVED] }),
);
const archived = build("archived", { SITE_ARCHIVE: ARCHIVED.version, SITE_METADATA_FILE: path.join(work, "archived.json") });
const site = build("site", { SITE_ARCHIVE: "", SITE_METADATA_FILE: path.join(work, "latest.json") });

const { versions } = JSON.parse(readFileSync(path.join(site, "versions.json"), "utf8"));
const root = versions.find((version) => version.latest).url;

// Doc pages, from the build: every page but the home page, the 404 and the archived lines.
const pages = (function walk(dir, rel = "") {
  return readdirSync(dir).flatMap((name) => {
    const full = path.join(dir, name);
    if (statSync(full).isDirectory()) return /^v\d+\.\d+$/.test(name) && !rel ? [] : walk(full, `${rel}${name}/`);
    return name === "index.html" && rel ? [rel] : [];
  });
})(site).sort();
const html = (page) => readFileSync(path.join(site, page, "index.html"), "utf8");
const codePage = pages.find((page) => html(page).includes('class="expressive-code'));
const testPage = codePage ?? pages[0];
const otherPage = pages.find((page) => page !== testPage);
const missingPage = pages.find((page) => page !== testPage && page !== otherPage);

cpSync(archived, path.join(site, ARCHIVED.version), { recursive: true });
rmSync(path.join(site, ARCHIVED.version, missingPage), { recursive: true, force: true });

// ------------------------------------------------------------------ server

const types = { ".html": "text/html", ".css": "text/css", ".js": "text/javascript", ".json": "application/json", ".svg": "image/svg+xml", ".png": "image/png", ".woff2": "font/woff2" };
const server = createServer((request, response) => {
  const url = new URL(request.url, "http://localhost");
  if (!url.pathname.startsWith(root)) return response.writeHead(404).end();
  let file = path.join(site, decodeURIComponent(url.pathname.slice(root.length)));
  if (existsSync(file) && statSync(file).isDirectory()) file = path.join(file, "index.html");
  if (!file.startsWith(site) || !existsSync(file)) return response.writeHead(404).end();
  response.writeHead(200, { "content-type": types[path.extname(file)] ?? "application/octet-stream" });
  if (request.method === "HEAD") return response.end();
  createReadStream(file).pipe(response);
});
await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
const origin = `http://127.0.0.1:${server.address().port}`;
const at = (page = "", line = "") => `${origin}${root}${line ? `${line}/` : ""}${page}`;

const browser = await puppeteer.launch({
  headless: true,
  args: process.platform === "linux" ? ["--no-sandbox"] : [],
});
const results = [];
const check = (name, ok, detail = "") => results.push(`${ok ? "PASS" : "FAIL"}  ${name}${ok || !detail ? "" : `  (${detail})`}`);
const wait = (ms = 150) => new Promise((resolve) => setTimeout(resolve, ms));

async function open(url, { width = 1440, height = 900, mobile = false, scheme = "dark", storage = true } = {}) {
  const page = await browser.newPage();
  page.errors = [];
  page.on("pageerror", (error) => page.errors.push(error.message));
  if (!storage) {
    await page.evaluateOnNewDocument(() => {
      for (const method of ["getItem", "setItem", "removeItem"]) {
        Storage.prototype[method] = () => {
          throw new DOMException("refused", "SecurityError");
        };
      }
    });
  }
  await page.emulateMediaFeatures([{ name: "prefers-color-scheme", value: scheme }]);
  await page.setViewport({ width, height, isMobile: mobile, hasTouch: mobile, deviceScaleFactor: mobile ? 2 : 1 });
  await page.goto(url, { waitUntil: "networkidle0" });
  return page;
}

// ------------------------------------------------------------------ theme

const themeState = (page) =>
  page.evaluate(() => {
    let stored;
    try {
      stored = localStorage.getItem("starlight-theme");
    } catch {
      stored = "<storage refused>";
    }
    const pressed = [...document.querySelectorAll("ns-theme-switch")].map((group) =>
      [...group.querySelectorAll("button")].filter((b) => b.getAttribute("aria-pressed") === "true").map((b) => b.dataset.value).join(","),
    );
    const menu = document.querySelector("ns-theme-menu");
    const button = menu.querySelector(".ns-theme-button");
    return {
      theme: document.documentElement.dataset.theme,
      stored,
      pressed,
      pref: menu.dataset.pref,
      checked: [...menu.querySelectorAll('[role="menuitemradio"]')].filter((i) => i.getAttribute("aria-checked") === "true").map((i) => i.dataset.value).join(","),
      icon: [...menu.querySelectorAll(".ns-theme-button svg")].filter((svg) => getComputedStyle(svg).display !== "none").map((svg) => svg.classList[0]).join(","),
      expanded: button.getAttribute("aria-expanded"),
      open: !menu.querySelector('[role="menu"]').hidden,
      focus: document.activeElement === button ? "button" : (document.activeElement?.dataset?.value ?? document.activeElement?.tagName),
    };
  });
const shows = (s, value) => s.pref === value && s.checked === value && s.icon === value && s.pressed.every((p) => p === value);
async function choose(page, value) {
  await page.click(".ns-theme-button");
  await wait();
  await page.click(`[role="menuitemradio"][data-value="${value}"]`);
  await wait();
  return themeState(page);
}
const codeBackground = (page) =>
  page.evaluate(() => {
    const pre = document.querySelector(".expressive-code pre");
    return pre && getComputedStyle(pre).backgroundColor;
  });

async function themeRun(label, { storage = true, scheme = "dark" } = {}) {
  const page = await open(at(testPage), { storage, scheme });
  const first = await themeState(page);
  check(`theme, ${label}: the first load follows the system (${scheme})`, first.theme === scheme && shows(first, "auto") && !first.open, JSON.stringify(first));
  await page.click(".ns-theme-button");
  await wait();
  let s = await themeState(page);
  check(`theme, ${label}: the button opens the menu on the chosen item`, s.open && s.expanded === "true" && s.focus === "auto", JSON.stringify(s));
  await page.click('[role="menuitemradio"][data-value="dark"]');
  await wait();
  s = await themeState(page);
  check(`theme, ${label}: Dark, and the menu closes back to the button`, s.theme === "dark" && shows(s, "dark") && !s.open && s.focus === "button" && (!storage || s.stored === "dark"), JSON.stringify(s));
  const darkCode = await codeBackground(page);
  s = await choose(page, "light");
  check(`theme, ${label}: Light`, s.theme === "light" && shows(s, "light") && (!storage || s.stored === "light"), JSON.stringify(s));
  if (codePage) {
    const lightCode = await codeBackground(page);
    check(`theme, ${label}: code stays on its dark panel in the light theme`, darkCode === lightCode, `${darkCode} vs ${lightCode}`);
  }
  s = await choose(page, "auto");
  check(`theme, ${label}: System`, s.theme === scheme && shows(s, "auto") && (!storage || s.stored === ""), JSON.stringify(s));
  await page.evaluate(() => document.querySelector('ns-theme-switch [data-value="dark"]').click());
  await wait();
  s = await themeState(page);
  check(`theme, ${label}: the phone menu's switch updates the header menu`, s.theme === "dark" && shows(s, "dark"), JSON.stringify(s));
  await choose(page, "auto");
  if (storage) {
    await choose(page, "light");
    await page.goto(at(otherPage), { waitUntil: "networkidle0" });
    s = await themeState(page);
    check(`theme, ${label}: Light survives a page change`, s.theme === "light" && shows(s, "light"), JSON.stringify(s));
    await page.emulateMediaFeatures([{ name: "prefers-color-scheme", value: "light" }]);
    await choose(page, "auto");
    await page.emulateMediaFeatures([{ name: "prefers-color-scheme", value: "dark" }]);
    await wait();
    s = await themeState(page);
    check(`theme, ${label}: System follows a live change of the system theme`, s.theme === "dark" && shows(s, "auto"), JSON.stringify(s));
  }
  check(`theme, ${label}: no script errors`, page.errors.length === 0, page.errors.join(" | "));
  await page.close();
}

await themeRun("storage allowed");
await themeRun("storage refused, dark system", { storage: false });
await themeRun("storage refused, light system", { storage: false, scheme: "light" });

{
  const page = await open(at(testPage));
  await page.focus(".ns-theme-button");
  await page.keyboard.press("ArrowDown");
  let s = await themeState(page);
  check("theme menu: ArrowDown opens it on the first item", s.open && s.focus === "light", JSON.stringify(s));
  await page.keyboard.press("End");
  await page.keyboard.press("ArrowDown");
  s = await themeState(page);
  check("theme menu: End, then ArrowDown wraps to the top", s.focus === "light", JSON.stringify(s));
  await page.keyboard.press("ArrowUp");
  await page.keyboard.press("ArrowUp");
  s = await themeState(page);
  check("theme menu: ArrowUp wraps to the bottom and moves up", s.focus === "dark", JSON.stringify(s));
  await page.keyboard.press("Enter");
  await wait();
  s = await themeState(page);
  check("theme menu: Enter chooses and returns to the button", s.theme === "dark" && !s.open && s.focus === "button", JSON.stringify(s));
  await page.keyboard.press("Enter");
  await wait();
  await page.keyboard.press("Escape");
  s = await themeState(page);
  check("theme menu: Escape closes it and returns to the button", !s.open && s.focus === "button", JSON.stringify(s));
  await page.keyboard.press("Space");
  await wait();
  await page.keyboard.press("Tab");
  s = await themeState(page);
  check("theme menu: Tab closes it", !s.open && s.expanded === "false", JSON.stringify(s));
  await page.click(".ns-theme-button");
  await wait();
  await page.mouse.click(700, 500);
  await wait();
  s = await themeState(page);
  check("theme menu: a click outside closes it without a change", !s.open && s.theme === "dark", JSON.stringify(s));
  await page.close();
}

{
  const page = await open(at(testPage), { width: 390, height: 844, mobile: true, scheme: "light" });
  const inBar = await page.evaluate(() => document.querySelector(".ns-theme-button").getBoundingClientRect().width);
  check("phone, doc page: no theme button in the bar", inBar === 0, `width ${inBar}`);
  await page.click(".sl-menu-button");
  await wait(300);
  const visible = await page.evaluate(() => [...document.querySelectorAll("ns-theme-switch")].filter((el) => el.getBoundingClientRect().width > 0).length);
  check("phone, doc page: the theme switch shows in the open menu", visible === 1, `visible switches: ${visible}`);
  await page.goto(at(), { waitUntil: "networkidle0" });
  const home = await page.evaluate(() => [...document.querySelectorAll(".ns-theme-button")].filter((el) => el.getBoundingClientRect().width > 0).length);
  check("phone, home page: the theme button shows in the bar", home === 1, `visible buttons: ${home}`);
  await page.close();
}

// ------------------------------------------------------------------ header and tabs

const versionMenu = (page) =>
  page.evaluate(() => {
    const menu = document.querySelector("ns-version-menu");
    const button = menu.querySelector(".ns-version");
    const rows = [...menu.querySelectorAll("li a")].map((a) => ({
      version: a.querySelector(".ns-line")?.textContent,
      release: a.querySelector(".ns-release")?.textContent,
      latest: a.querySelector(".ns-latest")?.textContent,
      current: a.getAttribute("aria-current") === "true",
      href: a.getAttribute("href"),
      columns: [".ns-line", ".ns-release", ".ns-tick"].map((cell) => Math.round(a.querySelector(cell)?.getBoundingClientRect().left ?? -1)),
    }));
    return { label: button.getAttribute("aria-label"), open: !menu.querySelector(".ns-versions").hidden, rows, focus: document.activeElement === button };
  });

{
  const page = await open(at(testPage));
  const layout = await page.evaluate(() => ({
    tab: Math.round(document.querySelector(".ns-tabs a").getBoundingClientRect().left),
    sidebar: Math.round(document.querySelector(".sidebar-pane").getBoundingClientRect().right),
    bar: Math.round(document.querySelector("header.header").getBoundingClientRect().height),
  }));
  check("header: one 64px row, with the first tab starting at the sidebar's edge", layout.bar === 64 && Math.abs(layout.tab - layout.sidebar) <= 1, JSON.stringify(layout));

  let m = await versionMenu(page);
  check("version menu: the chevron is named for the line", m.label === `Documentation version, ${LATEST.version}`, m.label);
  await page.click(".ns-version");
  await wait();
  m = await versionMenu(page);
  const [newest, older] = m.rows;
  check(
    "version menu: both lines, the latest marked and ticked",
    m.open && m.rows.length === 2 && newest.version === LATEST.version && newest.latest === "Latest" && newest.current && older.version === ARCHIVED.version && !older.current,
    JSON.stringify(m.rows),
  );
  check("version menu: the rows' columns line up", JSON.stringify(newest.columns) === JSON.stringify(older.columns), JSON.stringify(m.rows.map((r) => r.columns)));
  await page.keyboard.press("Escape");
  m = await versionMenu(page);
  check("version menu: Escape closes it and returns to the chevron", !m.open && m.focus, JSON.stringify(m));
  await page.click(".ns-version");
  await wait();
  await Promise.all([page.waitForNavigation({ waitUntil: "networkidle0" }), page.click(`li a[data-version="${ARCHIVED.version}"]`)]);
  check("version menu: switching opens the same page in the other line", new URL(page.url()).pathname === `${root}${ARCHIVED.version}/${testPage}`, page.url());
  check("header: no script errors", page.errors.length === 0, page.errors.join(" | "));
  await page.close();
}

{
  const page = await open(at());
  const tabs = await page.evaluate(() => [...document.querySelectorAll(".ns-tabs a")].map((a) => ({ label: a.textContent.trim(), href: a.getAttribute("href"), current: a.hasAttribute("aria-current") })));
  check("tabs: the home page sits outside every tab", tabs.length > 0 && tabs.every((tab) => !tab.current), JSON.stringify(tabs));
  const owned = [];
  for (const tab of tabs) {
    await page.goto(origin + tab.href, { waitUntil: "networkidle0" });
    const seen = await page.evaluate(() => ({
      current: document.querySelector('.ns-tabs a[aria-current="page"]')?.textContent.trim(),
      groups: [...document.querySelectorAll(".sidebar-content .ns-group .group-label .large")].map((label) => label.textContent.trim()),
    }));
    owned.push(...seen.groups);
    check(`tabs: ${tab.label} is current on its own page and has a sidebar of its own`, seen.current === tab.label && seen.groups.length > 0, JSON.stringify(seen));
  }
  check("tabs: no sidebar group appears under two tabs", new Set(owned).size === owned.length, JSON.stringify(owned));
  if (tabs.length > 1) {
    await page.goto(origin + tabs[0].href, { waitUntil: "networkidle0" });
    await page.evaluate(() => document.querySelector(".sidebar-content .ns-group summary").click());
    await wait();
    await page.goto(origin + tabs[1].href, { waitUntil: "networkidle0" });
    const open = await page.evaluate(() => document.querySelector(".sidebar-content .ns-group details").open);
    check("tabs: closing a group in one tab leaves the next tab's groups open", open === true, String(open));
  }
  await page.close();
}

// With many lines, the list scrolls inside the panel and the panel stays in the window. The rows
// are copies of the real ones, with the line being read moved to the bottom.
{
  const page = await open(at(testPage), { height: 600 });
  await page.evaluate(() => {
    const list = document.querySelector("ns-version-menu ul");
    const row = list.querySelector("li");
    for (let i = 0; i < 28; i++) list.append(row.cloneNode(true));
    list.querySelectorAll("a").forEach((a, i, all) => a.toggleAttribute("aria-current", i === all.length - 1));
    list.querySelector("li:last-child a").setAttribute("aria-current", "true");
  });
  await page.click(".ns-version");
  await wait();
  const fit = await page.evaluate(() => {
    const panel = document.querySelector(".ns-versions");
    const list = panel.querySelector("ul");
    const current = list.querySelector('[aria-current="true"]').getBoundingClientRect();
    const box = list.getBoundingClientRect();
    const last = [...panel.querySelectorAll(".ns-out")].at(-1).getBoundingClientRect();
    return {
      rows: list.querySelectorAll("li").length,
      scrolls: list.scrollHeight > list.clientHeight,
      inWindow: Math.round(panel.getBoundingClientRect().bottom) <= innerHeight,
      linksShown: last.bottom <= panel.getBoundingClientRect().bottom,
      currentShown: current.top >= box.top - 1 && current.bottom <= box.bottom + 1,
    };
  });
  check(
    "version menu: thirty lines scroll inside the panel, which stays in a 600px window",
    fit.rows === 30 && fit.scrolls && fit.inWindow && fit.linksShown,
    JSON.stringify(fit),
  );
  check("version menu: opening it scrolls the line being read into view", fit.currentShown, JSON.stringify(fit));
  await page.close();
}

// ------------------------------------------------------------------ an archived line

{
  const page = await open(at(testPage, ARCHIVED.version));
  await wait(300);
  let m = await versionMenu(page);
  check("archived line: the chevron is named for it", m.label === `Documentation version, ${ARCHIVED.version}`, m.label);
  await page.click(".ns-version");
  await wait();
  m = await versionMenu(page);
  check(
    "archived line: its menu is rebuilt from the root's versions.json",
    m.rows.length === 2 && m.rows[0].version === LATEST.version && m.rows[0].latest === "Latest" && !m.rows[0].current && m.rows[1].current && m.rows[0].href === root,
    JSON.stringify(m.rows),
  );
  await page.keyboard.press("Escape");

  const notice = () =>
    page.evaluate(() => {
      const element = document.querySelector("ns-archived-notice");
      return element && {
        text: element.querySelector("p").textContent.replace(/\s+/g, " ").trim(),
        href: element.querySelector("p a").getAttribute("href"),
        shown: !element.hidden && getComputedStyle(element).display !== "none",
        close: !element.querySelector(".ns-close").hidden,
      };
    });
  let n = await notice();
  check(
    "archived line: the notice names the latest release and links to this page in it",
    n?.text === `You're reading the docs for ${ARCHIVED.version}. The latest release is ${LATEST.release}. Read this page in ${LATEST.version}` && n.href === `${root}${testPage}` && n.shown && n.close,
    JSON.stringify(n),
  );
  const links = await page.evaluate((base) => [...document.querySelectorAll(`.sl-markdown-content a[href^="${base}"]`)].map((a) => a.getAttribute("href")), root);
  check("archived line: the page's links stay in the line", links.every((href) => href.startsWith(`${root}${ARCHIVED.version}/`)), links.join(" "));
  const pageEnd = await page.evaluate(() => [...document.querySelectorAll(".ns-page-end a")].map((a) => a.textContent.trim()));
  check("archived line: no edit link, but the report link stays", pageEnd.length === 1 && /report/i.test(pageEnd[0]), JSON.stringify(pageEnd));

  await page.click("ns-archived-notice .ns-close");
  await wait();
  n = await notice();
  const focus = await page.evaluate(() => document.activeElement?.id);
  check("archived line: closing the notice hides it and moves focus to the title", !n.shown && focus === "_top", JSON.stringify({ ...n, focus }));
  const key = `archived-notice:${root}${ARCHIVED.version}`;
  await page.goto(at(otherPage, ARCHIVED.version), { waitUntil: "networkidle0" });
  n = await notice();
  const stored = await page.evaluate((k) => localStorage.getItem(k), key);
  check("archived line: it stays closed on the line's other pages, stored at the latest release", !n.shown && stored === LATEST.release, JSON.stringify({ ...n, stored }));
  await page.evaluate((k) => localStorage.setItem(k, "v0.0.1"), key);
  await page.reload({ waitUntil: "networkidle0" });
  await wait(300);
  n = await notice();
  check("archived line: a newer release than the one it was closed at brings it back", n.shown, JSON.stringify(n));

  await page.goto(at(missingPage), { waitUntil: "networkidle0" });
  await page.click(".ns-version");
  await wait();
  await Promise.all([page.waitForNavigation({ waitUntil: "networkidle0" }), page.click(`li a[data-version="${ARCHIVED.version}"]`)]);
  check("archived line: switching to it from a page it doesn't have lands on its first page", new URL(page.url()).pathname === `${root}${ARCHIVED.version}/`, page.url());
  check("archived line: no script errors", page.errors.length === 0, page.errors.join(" | "));
  await page.close();
}

await browser.close();
server.close();
console.log(results.join("\n"));
const failed = results.filter((line) => line.startsWith("FAIL")).length;
console.log(`\n${results.length - failed} passed, ${failed} failed.`);
process.exit(failed ? 1 : 0);
