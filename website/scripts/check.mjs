import { readFileSync, existsSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const websiteRoot = path.resolve(__dirname, "..");

const requiredFiles = [
  "favicon.svg",
  "index.html",
  "main.js",
  "style.css",
  "vite.config.js",
  "public/og-image.svg",
  "public/og-image.png",
];

const requiredDomIds = [
  "site-home-link",
  "nav-github-link",
  "nav-releases-link",
  "nav-apt-link",
  "nav-container-link",
  "nav-docs-link",
  "hero-docs-link",
  "release-version",
  "release-commit",
  "release-date",
  "release-fingerprint",
  "release-highlights",
  "apt-command",
  "apt-fingerprint-row",
  "container-command",
  "install-apt-link",
  "install-container-link",
  "install-release-link",
  "install-source-link",
  "footer-release-link",
  "footer-apt-link",
  "footer-container-link",
  "footer-docs-link",
  "footer-version",
  "footer-commit",
];

const requiredEnvBindings = [
  "process.env.PUBLIC_SITE_URL",
  "process.env.PUBLIC_RELEASE_VERSION",
  "process.env.PUBLIC_COMMIT",
  "process.env.PUBLIC_BUILD_DATE",
  "process.env.PUBLIC_APT_FINGERPRINT",
  "process.env.PUBLIC_RELEASE_HIGHLIGHTS_JSON",
];

const errors = [];

function assert(condition, message) {
  if (!condition) {
    errors.push(message);
  }
}

function load(relativePath) {
  const fullPath = path.join(websiteRoot, relativePath);
  assert(existsSync(fullPath), `Missing required file: ${relativePath}`);
  return existsSync(fullPath) ? readFileSync(fullPath, "utf8") : "";
}

function checkNodeSyntax(relativePath) {
  const result = spawnSync(process.execPath, ["--check", relativePath], {
    cwd: websiteRoot,
    encoding: "utf8",
  });

  if (result.status !== 0) {
    const output = [result.stdout, result.stderr].filter(Boolean).join("\n").trim();
    errors.push(`Syntax check failed for ${relativePath}${output ? `\n${output}` : ""}`);
  }
}

function findRootRelativeLocalAssets(html) {
  const assetRefs = [...html.matchAll(/\b(?:src|href)=["']([^"']+)["']/g)].map((match) => match[1]);
  return assetRefs.filter((ref) => ref.startsWith("/") && !ref.startsWith("//"));
}

const indexHtml = load("index.html");
const mainJs = load("main.js");
const viteConfig = load("vite.config.js");

for (const relativePath of requiredFiles) {
  assert(existsSync(path.join(websiteRoot, relativePath)), `Missing required file: ${relativePath}`);
}

checkNodeSyntax("main.js");
checkNodeSyntax("vite.config.js");

assert(
  indexHtml.includes('<script type="module" src="./main.js"></script>'),
  "index.html must load the Vite entrypoint via ./main.js",
);

assert(
  indexHtml.includes('<link rel="icon" href="./favicon.svg" type="image/svg+xml" />'),
  "index.html must declare the SVG favicon via ./favicon.svg",
);

assert(
  mainJs.includes("import './style.css';") || mainJs.includes('import "./style.css";'),
  "main.js must import ./style.css so Vite owns stylesheet output",
);

assert(
  viteConfig.includes("base: './'") || viteConfig.includes('base: "./"'),
  "vite.config.js must keep base set to ./ for GitHub Pages subpath compatibility",
);

for (const envBinding of requiredEnvBindings) {
  assert(mainJs.includes(envBinding), `main.js must reference ${envBinding}`);
}

for (const domId of requiredDomIds) {
  assert(indexHtml.includes(`id="${domId}"`), `index.html is missing required id="${domId}"`);
}

for (const tabId of ["apt", "container"]) {
  assert(indexHtml.includes(`data-tab="${tabId}"`), `index.html is missing data-tab="${tabId}"`);
  assert(indexHtml.includes(`id="panel-${tabId}"`), `index.html is missing id="panel-${tabId}"`);
}

for (const installTab of ["apt", "container"]) {
  assert(indexHtml.includes(`data-install-tab="${installTab}"`), `index.html is missing data-install-tab="${installTab}"`);
}

for (const removedInstallTab of ["archives", "source"]) {
  assert(!indexHtml.includes(`data-tab="${removedInstallTab}"`), `index.html should not render a ${removedInstallTab} command tab`);
  assert(!indexHtml.includes(`id="panel-${removedInstallTab}"`), `index.html should not render panel-${removedInstallTab}`);
  assert(!indexHtml.includes(`data-install-tab="${removedInstallTab}"`), `index.html should not route links to the removed ${removedInstallTab} tab`);
}

assert(mainJs.includes("website-metadata.json"), "main.js must keep website-metadata.json support for Pages refreshes");
assert(mainJs.includes("blackhole-threats-archive-keyring.fingerprint.txt"), "main.js must render the APT fingerprint verification command");
assert(mainJs.includes("Types: deb deb-src"), "main.js must keep deb-src support in the APT install command");

const rootRelativeAssets = findRootRelativeLocalAssets(indexHtml);
assert(
  rootRelativeAssets.length === 0,
  `index.html contains root-relative local asset paths that break GitHub Pages subpath deploys: ${rootRelativeAssets.join(", ")}`,
);

assert(indexHtml.includes('href="#install"'), 'index.html must retain the in-page install anchor link');
assert(indexHtml.includes('id="install"'), 'index.html must retain the install section target');

const siteUrl = "https://netspeedy.github.io/blackhole-threats/";
const ogImageUrl = "https://netspeedy.github.io/blackhole-threats/og-image.png?v=1";
assert(indexHtml.includes(`<link rel="canonical" href="${siteUrl}" />`), "index.html must declare a canonical URL");
assert(indexHtml.includes('name="robots" content="index, follow, max-image-preview:large"'), "index.html must allow large image previews for social crawlers");
assert(indexHtml.includes(`property="og:url" content="${siteUrl}"`), "index.html must declare the og:url");
assert(indexHtml.includes(`property="og:image" content="${ogImageUrl}"`), "index.html must reference the absolute og:image URL");
assert(indexHtml.includes('property="og:image:width" content="1200"'), "index.html must declare the og:image width");
assert(indexHtml.includes('property="og:image:height" content="630"'), "index.html must declare the og:image height");
assert(indexHtml.includes('name="twitter:card" content="summary_large_image"'), "index.html must declare a large-image Twitter card");
assert(indexHtml.includes(`name="twitter:image" content="${ogImageUrl}"`), "index.html must reference the absolute twitter:image URL");

for (const requiredInstallFact of [
  "amd64",
  "arm64",
  "armhf",
  "deb-src",
  "linux/amd64",
  "linux/arm64",
  "linux/arm",
  "sha256sums.txt",
  "go 1.25.0",
]) {
  assert(indexHtml.includes(requiredInstallFact), `index.html install matrix must mention ${requiredInstallFact}`);
}

if (errors.length > 0) {
  console.error("Website validation failed:\n");
  for (const error of errors) {
    console.error(`- ${error}`);
  }
  process.exit(1);
}

console.log("Website validation passed.");
