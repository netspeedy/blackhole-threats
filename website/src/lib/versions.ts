// Browser helpers for the version menu and the archived-docs notice.
import type { DocsVersion } from "./site";

let list: Promise<DocsVersion[] | undefined> | undefined;

/** Every published docs line, from the root's versions.json, or undefined if it can't be read. */
export function loadVersions(root: string): Promise<DocsVersion[] | undefined> {
  list ??= fetch(`${root}versions.json`)
    .then((response) => (response.ok ? response.json() : undefined))
    .then((data) => (Array.isArray(data?.versions) ? (data.versions as DocsVersion[]) : undefined))
    .catch(() => undefined);
  return list;
}

/**
 * The page being read, in the line starting at `target`, if that line has it. Otherwise the
 * line's first page. `base` is where the line being read starts.
 */
export async function samePageIn(target: string, base: string): Promise<string> {
  const path = location.pathname.startsWith(base) ? location.pathname.slice(base.length) : "";
  if (!path) return target;
  const candidate = `${target}${path}`;
  try {
    const response = await fetch(candidate, { method: "HEAD" });
    return response.ok ? candidate : target;
  } catch {
    return target;
  }
}
