// The colour theme, shared by the header's theme menu and the three-way switch in the phone menu.
//
// It keeps Starlight's contract: the choice lives in localStorage under "starlight-theme"
// ("light", "dark", or empty for the system setting), and <html data-theme> holds the theme in
// use. Starlight's inline ThemeProvider script reads the same key before first paint, so pages
// never flash. Each control registers a callback here, so choosing in one updates the others.

export type Theme = "auto" | "dark" | "light";

const storageKey = "starlight-theme";
const listeners = new Set<(theme: Theme) => void>();

export const parseTheme = (value: unknown): Theme =>
  value === "dark" || value === "light" ? value : "auto";

export function loadTheme(): Theme {
  try {
    return parseTheme(localStorage.getItem(storageKey));
  } catch {
    return "auto";
  }
}

const systemTheme = () => (matchMedia("(prefers-color-scheme: light)").matches ? "light" : "dark");

export function applyTheme(theme: Theme): void {
  document.documentElement.dataset.theme = theme === "auto" ? systemTheme() : theme;
  try {
    localStorage.setItem(storageKey, theme === "auto" ? "" : theme);
  } catch {
    // Private windows can refuse storage; the theme still applies for this page.
  }
  listeners.forEach((listener) => listener(theme));
}

/** Calls `listener` with every theme chosen from now on. */
export function onThemeChange(listener: (theme: Theme) => void): void {
  listeners.add(listener);
}

matchMedia("(prefers-color-scheme: light)").addEventListener("change", () => {
  if (loadTheme() === "auto") applyTheme("auto");
});
