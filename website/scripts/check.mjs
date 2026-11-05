// Validate the built site in dist/. Run `npm run build` first.
//
// It checks that every page in src/content/docs was built, that search was indexed, that the
// icons and share card are the sizes link previews and browsers expect, that every page carries
// the share and icon tags, and that every link into the site resolves under the base GitHub
// Pages serves it from. The site root comes from the build's versions.json, where the latest line
// is listed at the root.
//
// With SITE_ARCHIVE set, dist/ is an archived release line built for <root><line>/, and its links
// to the root (the latest docs and the other lines) can't be checked from here. Without it, any
// archived lines the Pages workflow has unpacked into dist/<line>/ are checked with the rest.
import { existsSync, readdirSync, readFileSync, statSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const dist = path.join(root, "dist");
const docs = path.join(root, "src", "content", "docs");
const archive = process.env.SITE_ARCHIVE || undefined;
const errors = [];

const fail = (message) => errors.push(message);

function walk(dir, keep) {
  return readdirSync(dir).flatMap((name) => {
    const full = path.join(dir, name);
    return statSync(full).isDirectory() ? walk(full, keep) : keep(full) ? [full] : [];
  });
}

/** Width and height from a PNG's IHDR chunk. */
function pngSize(file) {
  const data = readFileSync(file);
  if (data.subarray(1, 4).toString() !== "PNG") return undefined;
  return [data.readUInt32BE(16), data.readUInt32BE(20)];
}

/** Sizes of the images in an ICO file. A stored 0 means 256. */
function icoSizes(file) {
  const data = readFileSync(file);
  if (data.readUInt16LE(0) !== 0 || data.readUInt16LE(2) !== 1) return undefined;
  return Array.from({ length: data.readUInt16LE(4) }, (_, i) => data[6 + 16 * i] || 256);
}

function checkAsset(name, check) {
  const file = path.join(dist, name);
  if (!existsSync(file)) return fail(`dist/${name} is missing`);
  check(file);
}

if (!existsSync(path.join(dist, "index.html"))) {
  console.error("dist/index.html is missing; run `npm run build` first.");
  process.exit(1);
}

// Where the site lives: the root from the latest line's entry in versions.json, and the host from
// the home page's canonical link, so the share card's absolute address can be checked.
const versionsFile = path.join(dist, "versions.json");
const siteRoot = existsSync(versionsFile)
  ? JSON.parse(readFileSync(versionsFile, "utf8")).versions?.find?.((version) => version.latest)?.url
  : undefined;
const canonical = /<link rel="canonical" href="([^"]+)"/.exec(readFileSync(path.join(dist, "index.html"), "utf8"))?.[1];
if (typeof siteRoot !== "string" || !siteRoot.startsWith("/") || !canonical) {
  console.error("dist/versions.json has no latest line, or dist/index.html has no canonical link; rebuild the site.");
  process.exit(1);
}
const card = `${new URL(canonical).origin}${siteRoot}og-image.png?v=`;
const base = archive ? `${siteRoot}${archive}/` : siteRoot;

// Every content page has a built route.
const pages = walk(docs, (file) => /\.mdx?$/.test(file)).map((file) => {
  const slug = path.relative(docs, file).replace(/\.mdx?$/, "").replace(/(^|\/)index$/, "");
  return slug ? `${slug}/index.html` : "index.html";
});
for (const page of pages) {
  if (!existsSync(path.join(dist, page))) fail(`dist/${page} wasn't built`);
}

// The page GitHub Pages serves for missing addresses, from src/pages/404.astro.
checkAsset("404.html", (file) => {
  if (!readFileSync(file, "utf8").includes("Page not found")) fail("dist/404.html isn't the not-found page");
});

// Search, icons and the share card.
checkAsset("pagefind/pagefind.js", () => {});
checkAsset("favicon.svg", (file) => {
  if (!readFileSync(file, "utf8").includes("<svg")) fail("dist/favicon.svg isn't an SVG");
});
checkAsset("favicon.ico", (file) => {
  const sizes = icoSizes(file);
  if (!sizes || ![16, 32, 48].every((size) => sizes.includes(size))) {
    fail(`dist/favicon.ico should hold 16, 32 and 48 px images, found ${sizes ?? "no icon"}`);
  }
});
checkAsset("apple-touch-icon.png", (file) => {
  const size = pngSize(file);
  if (size?.join("x") !== "180x180") fail(`dist/apple-touch-icon.png is ${size?.join("x")}, not 180x180`);
});
checkAsset("og-image.png", (file) => {
  const size = pngSize(file);
  if (size?.join("x") !== "1200x630") fail(`dist/og-image.png is ${size?.join("x")}, not 1200x630`);
});

// Every published line, and that each one the root lists is there.
checkAsset("versions.json", (file) => {
  const { versions } = JSON.parse(readFileSync(file, "utf8"));
  if (!Array.isArray(versions) || versions.length === 0) return fail("dist/versions.json lists no versions");
  for (const { version, url } of versions) {
    if (archive || typeof url !== "string" || !url.startsWith(siteRoot)) continue;
    if (!existsSync(path.join(dist, url.slice(siteRoot.length), "index.html"))) {
      fail(`dist/versions.json lists ${version} at ${url}, which wasn't published`);
    }
  }
});

// Every page's head, and every link into the site. A page in an unpacked archived line has that
// line's base.
const required = (pageBase) => [
  [`<link rel="shortcut icon" href="${pageBase}favicon.svg" type="image/svg+xml"`, "the SVG favicon"],
  [`<link rel="icon" href="${pageBase}favicon.ico" sizes="32x32"`, "the favicon.ico fallback"],
  [`<link rel="apple-touch-icon" href="${pageBase}apple-touch-icon.png"`, "the Apple touch icon"],
  [`<meta property="og:image" content="${card}`, "og:image with a ?v= cache key"],
  ['<meta property="og:image:type" content="image/png"', "og:image:type"],
  ['<meta property="og:image:width" content="1200"', "og:image:width"],
  ['<meta property="og:image:height" content="630"', "og:image:height"],
  ['<meta property="og:image:alt"', "og:image:alt"],
  ['<meta name="twitter:card" content="summary_large_image"', "a large twitter:card"],
  [`<meta name="twitter:image" content="${card}`, "twitter:image"],
];

for (const file of walk(dist, (f) => f.endsWith(".html"))) {
  const page = path.relative(dist, file);
  const html = readFileSync(file, "utf8");
  const line = !archive && /^(v\d+\.\d+)\//.exec(page)?.[1];
  for (const [needle, what] of required(line ? `${siteRoot}${line}/` : base)) {
    if (!html.includes(needle)) fail(`${page} is missing ${what}`);
  }
  for (const [, url] of html.matchAll(/\s(?:href|src)="([^"]+)"/g)) {
    if (!url.startsWith("/") || url.startsWith("//")) continue;
    if (!url.startsWith(base)) {
      if (archive && url.startsWith(siteRoot)) continue;
      fail(`${page} links to ${url}, which skips the ${base} base`);
      continue;
    }
    const target = decodeURIComponent(url.slice(base.length).replace(/[?#].*$/, ""));
    const resolved = path.join(dist, target);
    const exists = target === "" || existsSync(target.endsWith("/") ? path.join(resolved, "index.html") : resolved);
    if (!exists) fail(`${page} links to ${url}, which doesn't exist`);
  }
}

if (errors.length > 0) {
  console.error("Website validation failed:\n");
  for (const error of errors) console.error(`- ${error}`);
  process.exit(1);
}

console.log(`Website validation passed: ${pages.length} pages, icons, share card and links.`);
