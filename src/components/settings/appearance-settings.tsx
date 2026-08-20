import { useEffect, useState } from "react";
import { Moon, Sun, Monitor } from "lucide-react";
import { useTheme } from "next-themes";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { cn } from "@/lib/utils";

const COLOR_THEME_STORAGE = "app-color-theme";

export const COLOR_THEMES = [
  { id: "neutral", label: "Neutral", swatch: "bg-neutral-500" },
  { id: "blue", label: "Blue", swatch: "bg-blue-600" },
  { id: "red", label: "Red", swatch: "bg-red-600" },
  { id: "green", label: "Green", swatch: "bg-green-600" },
  { id: "yellow", label: "Yellow", swatch: "bg-yellow-400" },
  { id: "orange", label: "Orange", swatch: "bg-orange-500" },
  { id: "violet", label: "Violet", swatch: "bg-violet-600" },
  { id: "rose", label: "Rose", swatch: "bg-rose-600" },
] as const;

export type ColorThemeId = (typeof COLOR_THEMES)[number]["id"];

export function applyColorTheme(themeId: ColorThemeId) {
  const root = document.documentElement;
  if (themeId === "neutral") {
    delete root.dataset.colorTheme;
    localStorage.removeItem(COLOR_THEME_STORAGE);
  } else {
    root.dataset.colorTheme = themeId;
    localStorage.setItem(COLOR_THEME_STORAGE, themeId);
  }
}

export function readStoredColorTheme(): ColorThemeId {
  const raw = localStorage.getItem(COLOR_THEME_STORAGE);
  if (!raw) return "neutral";
  const match = COLOR_THEMES.find((t) => t.id === raw);
  return match?.id ?? "neutral";
}

export function AppearanceSettings() {
  const { theme, setTheme } = useTheme();
  const [colorTheme, setColorTheme] = useState<ColorThemeId>("neutral");
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    const stored = readStoredColorTheme();
    applyColorTheme(stored);
    setColorTheme(stored);
    setMounted(true);
  }, []);

  useEffect(() => {
    const onStorage = (e: StorageEvent) => {
      if (e.key === COLOR_THEME_STORAGE) {
        setColorTheme(readStoredColorTheme());
      }
    };
    window.addEventListener("storage", onStorage);
    return () => window.removeEventListener("storage", onStorage);
  }, []);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Appearance</CardTitle>
        <CardDescription>
          Choose light or dark mode and an accent palette for buttons and focus
          rings.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-8">
        <div className="space-y-3">
          <Label>Mode</Label>
          {mounted ? (
            <ToggleGroup
              type="single"
              variant="outline"
              spacing={0}
              value={theme ?? "system"}
              onValueChange={(v) => {
                if (v) setTheme(v);
              }}
            >
              <ToggleGroupItem value="light" aria-label="Light mode">
                <Sun />
                Light
              </ToggleGroupItem>
              <ToggleGroupItem value="dark" aria-label="Dark mode">
                <Moon />
                Dark
              </ToggleGroupItem>
              <ToggleGroupItem value="system" aria-label="System theme">
                <Monitor />
                System
              </ToggleGroupItem>
            </ToggleGroup>
          ) : (
            <div className="bg-muted h-9 w-56 max-w-full animate-pulse rounded-lg" />
          )}
        </div>

        <div className="space-y-3">
          <Label>Accent color</Label>
          <div className="flex flex-wrap gap-2">
            {COLOR_THEMES.map(({ id, label, swatch }) => (
              <button
                key={id}
                type="button"
                onClick={() => {
                  applyColorTheme(id);
                  setColorTheme(id);
                }}
                className={cn(
                  "flex items-center gap-2 rounded-lg border px-2.5 py-1.5 text-sm transition-colors",
                  colorTheme === id
                    ? "border-primary bg-accent"
                    : "border-transparent bg-muted/60 hover:bg-muted",
                )}
              >
                <span
                  className={cn(
                    "size-4 shrink-0 rounded-full ring-2 ring-background",
                    swatch,
                  )}
                  aria-hidden
                />
                {label}
              </button>
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
