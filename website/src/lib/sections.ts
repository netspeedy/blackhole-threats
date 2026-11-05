// The four sidebar groups each have a colour and an icon. The sidebar draws each group's tile
// from here (Sidebar.astro), the page's group colours its breadcrumb, step numbers, list markers,
// table of contents and pager (Head.astro sets --ns-page-* for that), and cards pointing into a
// group take its colour.
//
// The header's tabs each own some of the groups, and a page's sidebar only shows the groups of its
// tab (routeData.ts narrows it). Keep `sections` and `tabs` in step with the sidebar in
// astro.config.mjs.

export type Hue = "c1" | "c2" | "c3" | "c4";

export interface Section {
  label: string;
  hue: Hue;
  /** One line under the group's name in the home page's directory. */
  blurb: string;
  /** The inside of a 24x24 stroke icon. */
  icon: string;
}

export const sections: Section[] = [
  {
    label: "Getting started",
    hue: "c1",
    blurb: "Install it and run a first refresh.",
    icon: '<path d="M5 7l5 5-5 5"/><path d="M13 17h6"/>',
  },
  {
    label: "How it works",
    hue: "c2",
    blurb: "From feeds to communities to routes.",
    icon: '<path d="M3 5h18l-7 8v6l-4 2v-8z"/>',
  },
  {
    label: "Guides",
    hue: "c3",
    blurb: "Run it as a service and keep it healthy.",
    icon: '<path d="M3 6c3-1.3 6-1.3 9 .8 3-2.1 6-2.1 9-.8v13c-3-1.3-6-1.3-9 .8-3-2.1-6-2.1-9-.8z"/><path d="M12 6.8v13"/>',
  },
  {
    label: "Reference",
    hue: "c4",
    blurb: "Look something up.",
    icon:
      '<path d="M9 4H7.5A2.5 2.5 0 0 0 5 6.5V10l-2 2 2 2v3.5A2.5 2.5 0 0 0 7.5 20H9"/>' +
      '<path d="M15 4h1.5A2.5 2.5 0 0 1 19 6.5V10l2 2-2 2v3.5a2.5 2.5 0 0 1-2.5 2.5H15"/>',
  },
];

export interface Tab {
  label: string;
  /** The sidebar groups this tab owns, by label. */
  groups: string[];
}

export const tabs: Tab[] = [
  { label: "Documentation", groups: ["Getting started", "How it works", "Guides"] },
  { label: "Reference", groups: ["Reference"] },
];

interface Link {
  type: "link";
  label: string;
  href: string;
  isCurrent: boolean;
}
interface Group {
  type: "group";
  label: string;
  entries: Entry[];
}
type Entry = Link | Group;

const linksIn = (list: readonly Entry[]): Link[] =>
  list.flatMap((entry) => (entry.type === "link" ? [entry] : linksIn(entry.entries)));

/** The top-level group holding a link that passes `match`. */
function groupOf(entries: readonly Entry[], match: (link: Link) => boolean): Section | undefined {
  for (const entry of entries) {
    if (entry.type !== "group") continue;
    const holds = (list: readonly Entry[]): boolean =>
      list.some((child) => (child.type === "link" ? match(child) : holds(child.entries)));
    if (holds(entry.entries)) return sections.find((section) => section.label === entry.label);
  }
  return undefined;
}

/** Each sidebar group with its section and the links in it, in sidebar order. */
export function directory(sidebar: readonly Entry[]) {
  return sidebar.flatMap((entry) => {
    if (entry.type !== "group") return [];
    const section = sections.find((candidate) => candidate.label === entry.label);
    return section ? [{ section, links: linksIn(entry.entries) }] : [];
  });
}

/** Each tab with the page it opens on (its first group's first page) and whether it's current. */
export function tabsFor(sidebar: readonly Entry[]) {
  return tabs.map((tab) => {
    const groups = sidebar.filter((entry): entry is Group => entry.type === "group" && tab.groups.includes(entry.label));
    const links = groups.flatMap((group) => linksIn(group.entries));
    return { ...tab, href: links[0]?.href, current: links.some((link) => link.isCurrent) };
  });
}

/** The group's icon as a CSS mask image, for the sidebar's tiles. */
export function iconMask(section: Section): string {
  const svg =
    `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="#000" stroke-width="2.1" ` +
    `stroke-linecap="round" stroke-linejoin="round">${section.icon}</svg>`;
  return `url("data:image/svg+xml,${encodeURIComponent(svg)}")`;
}

/** The section of the page being rendered, or undefined for pages outside the sidebar. */
export const currentSection = (sidebar: readonly Entry[]) => groupOf(sidebar, (link) => link.isCurrent);

/** The section a link into the site lands in. */
export const sectionOfHref = (sidebar: readonly Entry[], href: string) =>
  groupOf(sidebar, (link) => link.href === href);
