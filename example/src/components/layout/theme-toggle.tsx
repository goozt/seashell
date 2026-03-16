"use client";

import { useEffect, useState } from "react";
import { Sun, Moon, Monitor } from "lucide-react";
import { Button } from "@/components/ui/button";
import { type Theme, applyTheme, getStoredTheme, setStoredTheme, nextTheme } from "@/lib/theme";

const ICONS: Record<Theme, React.ElementType> = {
  light: Sun,
  dark: Moon,
  system: Monitor,
};

const TITLES: Record<Theme, string> = {
  light: "Light theme",
  dark: "Dark theme",
  system: "System theme",
};

export function ThemeToggle() {
  const [theme, setTheme] = useState<Theme>("system");

  useEffect(() => {
    const stored = getStoredTheme();
    setTheme(stored);
    applyTheme(stored);
  }, []);

  useEffect(() => {
    if (theme !== "system") return;
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const handler = () => applyTheme("system");
    mq.addEventListener("change", handler);
    return () => mq.removeEventListener("change", handler);
  }, [theme]);

  function handleClick() {
    const next = nextTheme(theme);
    setTheme(next);
    setStoredTheme(next);
    applyTheme(next);
  }

  const Icon = ICONS[theme];

  return (
    <Button
      variant="ghost"
      size="icon"
      onClick={handleClick}
      className="h-8 w-8"
      title={TITLES[theme]}
    >
      <Icon className="h-4 w-4" />
    </Button>
  );
}
