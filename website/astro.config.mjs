import { unified } from "@astrojs/markdown-remark";
import starlight from "@astrojs/starlight";
import { defineConfig, passthroughImageService } from "astro/config";

import { codeTheme } from "./src/lib/code-theme.mjs";
import rehypeArchiveLinks from "./src/lib/rehype-archive-links.mjs";
import rehypeCodeWords from "./src/lib/rehype-code-words.mjs";

const site = "https://netspeedy.github.io";
const root = "/blackhole-threats";
const repo = "https://github.com/netspeedy/blackhole-threats";

// The latest docs are built at the root. An archived release line is built with SITE_ARCHIVE set
// to the line, such as v1.1, which puts it at /blackhole-threats/v1.1/ (see src/lib/site.ts).
const archive = process.env.SITE_ARCHIVE || undefined;
if (archive && !/^v\d+\.\d+$/.test(archive)) {
  throw new Error(`SITE_ARCHIVE must be a release line such as v0.8, not ${archive}`);
}
const base = archive ? `${root}/${archive}` : root;

// The share card is written by scripts/readme/build_readme_hero.py. Bump ?v= whenever it
// changes so link previews fetch the new one. Archived lines share the root's card.
const card = `${site}${root}/og-image.png?v=2`;
const cardAlt =
  "blackhole-threats share card: Threat feeds in. Blackhole routes out. Beside it, text, JSON " +
  "and local file feeds spiral into a black hole while jets carry announcements and withdrawals out.";

export default defineConfig({
  site,
  base,
  trailingSlash: "always",
  markdown: {
    processor: unified({
      rehypePlugins: [
        rehypeCodeWords,
        ...(archive ? [[rehypeArchiveLinks, { root: `${root}/`, base: `${base}/` }]] : []),
      ],
    }),
  },
  // No page uses Astro's image pipeline, so skip sharp.
  image: { service: passthroughImageService() },
  integrations: [
    starlight({
      title: "blackhole-threats",
      description:
        "Turns threat feeds into BGP blackhole routes: it reads text and JSON feeds, summarises the " +
        "prefixes and announces them to your routers with a community they can drop on.",
      favicon: "/favicon.svg",
      // An archived line's pages are copies of an older main, so there's nothing to edit.
      editLink: archive ? {} : { baseUrl: `${repo}/edit/main/website/` },
      // Pages build from a shallow clone, where every file would show the same commit date.
      lastUpdated: false,
      // The 404 page is src/pages/404.astro; see there for why it isn't a 404.md.
      disable404Route: true,
      credits: false,
      customCss: [
        "@fontsource-variable/inter",
        "@fontsource-variable/jetbrains-mono",
        "./src/styles/theme.css",
      ],
      routeMiddleware: "./src/routeData.ts",
      components: {
        Banner: "./src/components/Banner.astro",
        Footer: "./src/components/Footer.astro",
        Head: "./src/components/Head.astro",
        Header: "./src/components/Header.astro",
        PageTitle: "./src/components/PageTitle.astro",
        Sidebar: "./src/components/Sidebar.astro",
        SiteTitle: "./src/components/SiteTitle.astro",
        ThemeProvider: "./src/components/ThemeProvider.astro",
        ThemeSelect: "./src/components/ThemeSelect.astro",
      },
      head: [
        { tag: "link", attrs: { rel: "icon", href: `${base}/favicon.ico`, sizes: "32x32" } },
        { tag: "link", attrs: { rel: "apple-touch-icon", href: `${base}/apple-touch-icon.png` } },
        {
          tag: "meta",
          attrs: { name: "theme-color", content: "#0A0D12", media: "(prefers-color-scheme: dark)" },
        },
        {
          tag: "meta",
          attrs: { name: "theme-color", content: "#FFFFFF", media: "(prefers-color-scheme: light)" },
        },
        { tag: "meta", attrs: { property: "og:image", content: card } },
        { tag: "meta", attrs: { property: "og:image:secure_url", content: card } },
        { tag: "meta", attrs: { property: "og:image:type", content: "image/png" } },
        { tag: "meta", attrs: { property: "og:image:width", content: "1200" } },
        { tag: "meta", attrs: { property: "og:image:height", content: "630" } },
        { tag: "meta", attrs: { property: "og:image:alt", content: cardAlt } },
        { tag: "meta", attrs: { name: "twitter:image", content: card } },
        { tag: "meta", attrs: { name: "twitter:image:alt", content: cardAlt } },
      ],
      tableOfContents: { minHeadingLevel: 2, maxHeadingLevel: 3 },
      expressiveCode: {
        // Code is always on a dark panel, in both site themes, so there's one theme and Starlight
        // neither swaps it nor recolours its frames from the page.
        themes: [codeTheme],
        useStarlightDarkModeSwitch: false,
        useStarlightUiThemeColors: false,
        // Shell commands read better without an empty terminal title bar. Blocks with a title,
        // like "Output" or a file name, keep their tab.
        defaultProps: { overridesByLang: { "bash,sh,shell": { frame: "none" } } },
        styleOverrides: {
          borderRadius: "12px",
          borderColor: "#1c2330",
          codeBackground: "#0b0f16",
          codeForeground: "#dce4ee",
          codeFontFamily: "var(--__sl-font-mono)",
          codeFontSize: "0.8125rem",
          codeLineHeight: "1.75",
          uiFontFamily: "var(--__sl-font)",
          uiFontSize: "0.8125rem",
          focusBorder: "#4bd8ff",
          scrollbarThumbColor: "#2a323d",
          scrollbarThumbHoverColor: "#3a4452",
          frames: {
            shadowColor: "transparent",
            editorTabBarBackground: "#0f141d",
            editorActiveTabBackground: "#0f141d",
            editorActiveTabForeground: "#e8ecf1",
            editorActiveTabIndicatorTopColor: "transparent",
            editorActiveTabIndicatorBottomColor: "#4bd8ff",
            editorTabBarBorderBottomColor: "#1c2330",
            editorBackground: "#0b0f16",
            terminalBackground: "#0b0f16",
            terminalTitlebarBackground: "#0f141d",
            terminalTitlebarForeground: "#8f9bab",
            terminalTitlebarBorderBottomColor: "#1c2330",
            terminalTitlebarDotsOpacity: "0",
            inlineButtonBackground: "#141b26",
            inlineButtonForeground: "#c3ccd8",
            inlineButtonBorder: "#273042",
            tooltipSuccessBackground: "#1c7c4d",
          },
        },
      },
      sidebar: [
        {
          label: "Getting started",
          items: ["introduction", "quickstart", "installation"],
        },
        {
          label: "How it works",
          items: [{ label: "Overview", slug: "how-it-works" }, "how-it-works/feeds", "how-it-works/bgp"],
        },
        {
          label: "Guides",
          items: [
            "guides/configuration",
            "guides/debian",
            "guides/container",
            "guides/router-policy",
            "guides/operations",
            "guides/troubleshooting",
          ],
        },
        {
          label: "Reference",
          items: [
            "reference/cli",
            "reference/configuration",
            "reference/feed-formats",
            "reference/logs",
            "reference/signals-and-exit-codes",
            "reference/files",
          ],
        },
      ],
    }),
  ],
});
