// Keep each word of inline code on one line.
//
// Browsers may break a line after any hyphen, so `--force` can end up as "-" on one line and
// "-force" on the next. Putting the whole code element in white-space: nowrap stops that, but
// then a long command runs off a phone screen. Instead, wrap each space-separated word in a
// span that doesn't break, so code still wraps at its spaces. The text is unchanged, so copying
// it gives exactly what was written.
export default function rehypeCodeWords() {
  const walk = (node, inPre) => {
    if (node.type !== "element" && node.type !== "root") return;
    const pre = inPre || (node.type === "element" && node.tagName === "pre");
    if (!pre && node.type === "element" && node.tagName === "code") {
      const [only] = node.children;
      if (node.children.length === 1 && only.type === "text") {
        node.children = only.value
          .split(/( +)/)
          .filter(Boolean)
          .map((part) =>
            part.trim()
              ? {
                  type: "element",
                  tagName: "span",
                  properties: { className: ["ns-word"] },
                  children: [{ type: "text", value: part }],
                }
              : { type: "text", value: part },
          );
      }
      return;
    }
    for (const child of node.children ?? []) walk(child, pre);
  };
  return (tree) => walk(tree, false);
}
