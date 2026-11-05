// Each header tab has its own sidebar, so a page's sidebar keeps only the groups of its tab. Pages
// outside every tab, such as the home page and the 404, keep the whole sidebar. The whole one stays
// on Astro.locals.fullSidebar for the tabs, the home page's cards and cards that link across tabs.
import { defineRouteMiddleware } from "@astrojs/starlight/route-data";

import { tabsFor } from "./lib/sections";

export const onRequest = defineRouteMiddleware((context) => {
  const route = context.locals.starlightRoute;
  context.locals.fullSidebar = route.sidebar;
  const tab = tabsFor(route.sidebar).find((candidate) => candidate.current);
  if (tab) {
    route.sidebar = route.sidebar.filter((entry) => entry.type === "group" && tab.groups.includes(entry.label));
  }
});
