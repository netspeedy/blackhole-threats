declare namespace App {
  interface Locals {
    /** The whole sidebar, before routeData.ts narrows it to the current tab. */
    fullSidebar: import("@astrojs/starlight/route-data").StarlightRouteData["sidebar"];
  }
}
