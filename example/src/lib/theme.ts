export type Theme = "light" | "dark" | "system";

export const THEME_KEY = "theme";
export const THEMES: Theme[] = ["light", "dark", "system"];

// ── Design Tokens (single source of truth) ────────────────────────────────────

export const colors = {
  // Dark palette (reference)
  backgroundDark: "#081212",   // deepest bg
  surfaceDark: "#102122",      // card / raised surface
  primary: "#0ddff2",          // teal/cyan accent
  primaryDark: "#0bbdcc",      // slightly darker for light-mode contrast
  // Light palette (complementary)
  backgroundLight: "#f3f9f9",  // very light teal-white
  surfaceLight: "#e8f4f4",     // light teal card surface
} as const;

// HSL equivalents used in globals.css (for shadcn CSS-variable system):
//   backgroundDark  → 182 88% 4%
//   surfaceDark     → 183 36% 10%
//   primary         → 184 93% 50%
//   primaryDark     → 184 72% 43%
//   backgroundLight → 180 30% 97%
//   surfaceLight    → 180 28% 93%

export const fonts = {
  display: ["Inter", "sans-serif"],
  mono: ["JetBrains Mono", "monospace"],
} as const;

export const radius = {
  sm: "0.375rem",
  default: "0.5rem",
  lg: "1rem",
  xl: "1.5rem",
} as const;

// ── Theme utilities ────────────────────────────────────────────────────────────

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
