// The code panels' colours. Code stays dark in both site themes, so this is the only Expressive
// Code theme. The hues are the README banner's (scripts/readme/build_readme_hero.py): cyan for
// flags, the pale amber for arguments, green for strings, the light indigo for keys and keywords,
// amber for numbers and literals.
//
// The scopes are the ones Shiki's shellscript and JSON grammars actually emit; check them with
// codeToTokens(..., { includeExplanation: true }) before adding more.
const ink = "#dce4ee";
const muted = "#7d8999";
const comment = "#6b7686";
const command = "#ffffff";
const flag = "#7fe3ff";
const argument = "#fde1a7";
const string = "#6fd8a5";
const key = "#a5b1ff";
const literal = "#f5b841";

export const codeTheme = {
  name: "ns-code",
  type: "dark",
  colors: {
    "editor.background": "#0b0f16",
    "editor.foreground": ink,
    "editor.selectionBackground": "#1d3a52",
  },
  tokenColors: [
    { settings: { foreground: ink } },
    { scope: ["comment", "punctuation.definition.comment"], settings: { foreground: comment, fontStyle: "italic" } },
    { scope: ["entity.name.command", "entity.name.function.call"], settings: { foreground: command, fontStyle: "bold" } },
    { scope: ["constant.other.option"], settings: { foreground: flag } },
    { scope: ["string.unquoted.argument"], settings: { foreground: argument } },
    { scope: ["string", "string.quoted"], settings: { foreground: string } },
    { scope: ["support.type.property-name", "entity.name.tag", "keyword", "storage"], settings: { foreground: key } },
    { scope: ["constant.numeric", "constant.language"], settings: { foreground: literal } },
    { scope: ["variable", "variable.other.assignment"], settings: { foreground: ink } },
    { scope: ["punctuation", "keyword.operator", "constant.character.escape.line-continuation"], settings: { foreground: muted } },
    // Quotes take the colour of what they quote, not the muted punctuation colour.
    { scope: ["punctuation.definition.string"], settings: { foreground: string } },
    { scope: ["punctuation.support.type.property-name"], settings: { foreground: key } },
  ],
};
