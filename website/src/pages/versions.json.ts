// Every docs line the site publishes, newest first. Archived lines read the root's copy when they
// load, so their version menus list lines that shipped after them.
import type { APIRoute } from "astro";

import { docs } from "../lib/site";

export const GET: APIRoute = () =>
  new Response(`${JSON.stringify({ versions: docs.versions }, null, 2)}\n`, {
    headers: { "content-type": "application/json" },
  });
