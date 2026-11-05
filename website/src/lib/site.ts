// Release details for the pages, read once at build time.
//
// The Pages workflow resolves the latest release (and the APT signing key, for a project that
// publishes an APT repository) before it builds the site, writes them to a JSON file and passes
// its path in SITE_METADATA_FILE. Local and pull request builds don't have that file, so they fall
// back to the newest local tag and leave the fingerprint out rather than guess it.
//
// The docs are versioned by release line (v0.8, v0.9 and so on). The latest line is built at the
// site root. When a newer line ships, the release workflow builds the old one again with
// SITE_ARCHIVE set to its line, which moves it under the root as <root>v0.8/, and the Pages
// workflow puts each archived line back beside the latest one and lists them in docs_versions.
import { execFileSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";

export const links = {
  org: "https://github.com/netspeedy",
  repo: "https://github.com/netspeedy/blackhole-threats",
  releases: "https://github.com/netspeedy/blackhole-threats/releases",
  issues: "https://github.com/netspeedy/blackhole-threats/issues",
  container: "https://github.com/netspeedy/blackhole-threats/pkgs/container/blackhole-threats",
  example: "https://github.com/netspeedy/blackhole-threats/blob/main/examples/blackhole-threats.yaml",
  siteUrl: "https://netspeedy.github.io/blackhole-threats/",
  // The APT repository shares the Pages root with the site, so these are absolute: they're
  // published beside the build, not in it, and website-check only follows links into the build.
  aptUrl: "https://netspeedy.github.io/blackhole-threats/",
  aptKey: "https://netspeedy.github.io/blackhole-threats/blackhole-threats-archive-keyring.gpg",
  aptKeyFingerprint: "https://netspeedy.github.io/blackhole-threats/blackhole-threats-archive-keyring.fingerprint.txt",
  image: "ghcr.io/netspeedy/blackhole-threats",
};

export interface Release {
  /** The release tag, such as v0.8.38, or "dev" when no tag is known. */
  version: string;
  /** True when version is a real tag rather than the "dev" placeholder. */
  tagged: boolean;
  /** The APT signing key's fingerprint in groups of four, only when the repository is published. */
  aptFingerprint?: string;
  /** The release page, or the releases list when there's no tag. */
  url: string;
}

interface Metadata {
  latest_release?: { tag_name?: string; html_url?: string } | null;
  apt_repository?: { available?: boolean; fingerprint?: string };
  /** The archived release lines published beside this build, each with its newest release. */
  docs_versions?: { version: string; release: string }[];
}

function readMetadata(): Metadata | undefined {
  const file = process.env.SITE_METADATA_FILE;
  if (!file) return undefined;
  if (!existsSync(file)) throw new Error(`SITE_METADATA_FILE points at ${file}, which doesn't exist`);
  return JSON.parse(readFileSync(file, "utf8")) as Metadata;
}

function latestTag(): string | undefined {
  try {
    const tag = execFileSync("git", ["describe", "--tags", "--abbrev=0", "--match", "v*"], {
      encoding: "utf8",
      stdio: ["ignore", "pipe", "ignore"],
    }).trim();
    return tag || undefined;
  } catch {
    return undefined;
  }
}

const metadata = readMetadata();

function resolveRelease(): Release {
  const latest = metadata?.latest_release ?? undefined;
  const tag = latest?.tag_name || latestTag();
  const apt = metadata?.apt_repository;
  return {
    version: tag ?? "dev",
    tagged: Boolean(tag),
    aptFingerprint: apt?.available && apt.fingerprint ? apt.fingerprint : undefined,
    url: latest?.html_url || (tag ? `${links.releases}/tag/${tag}` : links.releases),
  };
}

export const release = resolveRelease();

/** The release line a version belongs to: v0.8.38 is in v0.8. */
export const lineOf = (version: string) => /^v\d+\.\d+/.exec(version)?.[0];

export interface DocsVersion {
  /** The release line, such as v0.8, or "dev" for an untagged build. */
  version: string;
  /** The line's newest release, such as v0.8.38. */
  release: string;
  /** Where the line's docs start, such as <root>v0.8/. */
  url: string;
  /** The newest release's notes. */
  notes: string;
  latest: boolean;
}

const archive = process.env.SITE_ARCHIVE || undefined;
const base = import.meta.env.BASE_URL.replace(/\/?$/, "/");

/** The site root every line lives under, also in an archived build. */
export const siteRoot = archive ? base.slice(0, base.length - archive.length - 1) : base;

/**
 * A link from the root, such as <root>quickstart/, as a link to that page in the line being
 * built. In the latest line it's unchanged. rehype-archive-links.mjs does the same for the pages.
 */
export const docsHref = (href: string) =>
  archive && href.startsWith(siteRoot) && !/^v\d+\.\d+\//.test(href.slice(siteRoot.length))
    ? base + href.slice(siteRoot.length)
    : href;

function resolveDocs() {
  const current = archive ?? lineOf(release.version) ?? "dev";
  const self: DocsVersion = {
    version: current,
    release: release.version,
    url: base,
    notes: release.url,
    latest: !archive,
  };
  // An archived build only knows itself. Its version menu fetches the full list from the root's
  // versions.json, and this link to the latest docs stands in until it does.
  if (archive) {
    const latest: DocsVersion = { version: "Latest", release: "", url: siteRoot, notes: links.releases, latest: true };
    return { current, archived: true, versions: [latest, self] };
  }
  const older = (metadata?.docs_versions ?? [])
    .filter(({ version }) => version !== current)
    .sort((a, b) => b.version.localeCompare(a.version, "en", { numeric: true }))
    .map(({ version, release: tag }) => ({
      version,
      release: tag,
      url: `${siteRoot}${version}/`,
      notes: `${links.releases}/tag/${tag}`,
      latest: false,
    }));
  return { current, archived: false, versions: [self, ...older] };
}

/** Which line these docs are, whether it's an archived one, and every line published. */
export const docs = resolveDocs();

// ------------------------------------------------------------------ this project's copy
//
// What the shared components say about this project. The components themselves are the same in
// every netspeedy site, so a project's name, description and footer links live here.

export const product = {
  name: "blackhole-threats",
  description:
    "Turns threat feeds into BGP blackhole routes: it reads text and JSON feeds, summarises the " +
    "prefixes and announces them to your routers with a community they can drop on.",
};

/** The line under "Browse the docs" on the home page. */
export const directorySubtitle = "From a first refresh to every config key.";

/** The site footer's link columns. */
export const footerColumns = [
  {
    title: "Docs",
    items: [
      { label: "Introduction", href: `${base}introduction/` },
      { label: "Quickstart", href: `${base}quickstart/` },
      { label: "How it works", href: `${base}how-it-works/` },
      { label: "CLI reference", href: `${base}reference/cli/` },
    ],
  },
  {
    title: "Project",
    items: [
      { label: "GitHub", href: links.repo },
      { label: "Releases", href: links.releases },
      { label: "Issues", href: links.issues },
      { label: "Example config", href: links.example },
    ],
  },
  {
    title: "Install",
    items: [
      { label: "APT repository", href: `${base}installation/#signed-apt-repository` },
      { label: "GHCR image", href: links.container },
      { label: "Release archives", href: `${base}installation/#release-archives` },
      { label: "From source", href: `${base}installation/#from-source` },
    ],
  },
];
