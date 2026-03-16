export type Theme = "light" | "dark" | "system";

export const THEME_KEY = "theme";
export const THEMES: Theme[] = ["light", "dark", "system"];

export function getStoredTheme(): Theme {
  if (typeof window === "undefined") return "system";
  return (localStorage.getItem(THEME_KEY) as Theme) ?? "system";
}

export function setStoredTheme(theme: Theme): void {
  localStorage.setItem(THEME_KEY, theme);
}

export function resolveTheme(theme: Theme): "light" | "dark" {
  if (theme !== "system") return theme;
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

export function applyTheme(theme: Theme): void {
  document.documentElement.classList.toggle("dark", resolveTheme(theme) === "dark");
}

export function nextTheme(current: Theme): Theme {
  const idx = THEMES.indexOf(current);
  return THEMES[(idx + 1) % THEMES.length];
}
