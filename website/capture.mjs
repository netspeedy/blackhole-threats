// Screenshot the built site in both themes at desktop and phone widths.
//
// Run `npm run build` first. This serves dist/ under the site root GitHub Pages uses, and
// writes the PNGs to .captures/. Pass page paths to capture something other than the defaults,
// for example `npm run capture -- reference/cli/`.
import { createReadStream, existsSync, readFileSync, statSync } from "node:fs";
import { mkdir } from "node:fs/promises";
import { createServer } from "node:http";
import path from "node:path";
import { fileURLToPath } from "node:url";

import puppeteer from "puppeteer";

const root = path.dirname(fileURLToPath(import.meta.url));
const dist = path.join(root, "dist");
const outputDir = path.join(root, ".captures");
if (!existsSync(path.join(dist, "versions.json"))) {
  console.error("dist/versions.json is missing; run `npm run build` first.");
  process.exit(1);
}
// The latest line is listed at the site root.
const base = JSON.parse(readFileSync(path.join(dist, "versions.json"), "utf8")).versions.find((version) => version.latest).url;

const pages = process.argv.slice(2).length ? process.argv.slice(2) : ["", "quickstart/", "reference/cli/"];
const themes = ["light", "dark"];
const viewports = [
  { name: "desktop", width: 1440, height: 900, deviceScaleFactor: 1 },
  { name: "mobile", width: 390, height: 844, deviceScaleFactor: 2, isMobile: true, hasTouch: true },
];

const types = {
  ".html": "text/html; charset=utf-8",
  ".css": "text/css",
  ".js": "text/javascript",
  ".json": "application/json",
  ".svg": "image/svg+xml",
  ".png": "image/png",
  ".ico": "image/x-icon",
  ".woff2": "font/woff2",
  ".xml": "application/xml",
};

function serve(request, response) {
  const url = new URL(request.url, "http://localhost");
  if (!url.pathname.startsWith(base)) {
    response.writeHead(404).end();
    return;
  }
  let file = path.join(dist, decodeURIComponent(url.pathname.slice(base.length)));
  if (!file.startsWith(dist)) {
    response.writeHead(403).end();
    return;
  }
  if (existsSync(file) && statSync(file).isDirectory()) file = path.join(file, "index.html");
  if (!existsSync(file)) {
    response.writeHead(404).end();
    return;
  }
  response.writeHead(200, { "content-type": types[path.extname(file)] ?? "application/octet-stream" });
  createReadStream(file).pipe(response);
}

async function main() {
  if (!existsSync(path.join(dist, "index.html"))) {
    throw new Error("dist/index.html is missing; run `npm run build` first");
  }
  await mkdir(outputDir, { recursive: true });
  const server = createServer(serve);
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  const origin = `http://127.0.0.1:${server.address().port}`;
  const browser = await puppeteer.launch({
    headless: true,
    args: process.platform === "linux" ? ["--no-sandbox"] : [],
  });

  try {
    for (const page of pages) {
      for (const theme of themes) {
        for (const viewport of viewports) {
          const tab = await browser.newPage();
          await tab.emulateMediaFeatures([{ name: "prefers-reduced-motion", value: "reduce" }]);
          await tab.evaluateOnNewDocument((value) => localStorage.setItem("starlight-theme", value), theme);
          await tab.setViewport(viewport);
          await tab.goto(`${origin}${base}${page}`, { waitUntil: "networkidle0" });
          await tab.evaluate(() => document.fonts.ready);
          const name = `${page.replace(/\/$/, "").replaceAll("/", "-") || "home"}-${theme}-${viewport.name}.png`;
          await tab.screenshot({ path: path.join(outputDir, name), fullPage: true });
          console.log(path.join(".captures", name));
          await tab.close();
        }
      }
    }
  } finally {
    await browser.close();
    server.close();
  }
}

main().catch((error) => {
  console.error(error instanceof Error ? error.message : String(error));
  process.exitCode = 1;
});
