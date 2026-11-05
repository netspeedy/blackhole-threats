// In an archived line, keep the pages' links inside that line.
//
// The pages link to each other from the site root, as <root>guides/configuration/. When a line is
// built under <root>v0.8/, those would jump to the latest docs, so this points them at the same
// page in the line being built. Links that already name a line, and links off the site, are left.
export default function rehypeArchiveLinks({ root, base }) {
  const walk = (node) => {
    if (node.type === "element" && node.tagName === "a") {
      const { href } = node.properties ?? {};
      if (typeof href === "string" && href.startsWith(root) && !/^v\d+\.\d+\//.test(href.slice(root.length))) {
        node.properties.href = base + href.slice(root.length);
      }
    }
    node.children?.forEach(walk);
  };
  return (tree) => walk(tree);
}
