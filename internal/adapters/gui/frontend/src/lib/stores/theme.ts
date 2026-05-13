import { writable } from 'svelte/store';

export const THEME_STORAGE_KEY = 'llm-budget-tracker-theme';
export const themes = ['system', 'dark', 'light'] as const;

export type ThemeMode = (typeof themes)[number];
export type ResolvedTheme = Exclude<ThemeMode, 'system'>;

const DEFAULT_THEME_MODE: ThemeMode = 'system';
const DEFAULT_RESOLVED_THEME: ResolvedTheme = 'dark';

function isThemeMode(value: unknown): value is ThemeMode {
  return value === 'system' || value === 'dark' || value === 'light';
}

function canUseBrowserApis() {
  return typeof window !== 'undefined' && typeof document !== 'undefined';
}

function getStoredThemeMode(): ThemeMode | null {
  if (!canUseBrowserApis()) {
    return null;
  }

  try {
    const storedTheme = window.localStorage?.getItem(THEME_STORAGE_KEY);
    return isThemeMode(storedTheme) ? storedTheme : null;
  } catch {
    return null;
  }
}

function getSystemTheme(): ResolvedTheme {
  if (!canUseBrowserApis()) {
    return DEFAULT_RESOLVED_THEME;
  }

  if (window.matchMedia?.('(prefers-color-scheme: light)').matches) {
    return 'light';
  }

  if (window.matchMedia?.('(prefers-color-scheme: dark)').matches) {
    return 'dark';
  }

  return DEFAULT_RESOLVED_THEME;
}

function resolveThemeMode(themeMode: ThemeMode): ResolvedTheme {
  return themeMode === 'system' ? getSystemTheme() : themeMode;
}

function applyResolvedTheme(theme: ResolvedTheme) {
  if (!canUseBrowserApis()) {
    return;
  }

  document.documentElement.classList.toggle('dark', theme === 'dark');
  document.documentElement.classList.toggle('light', theme === 'light');

}

function persistThemeMode(themeMode: ThemeMode) {
  if (!canUseBrowserApis()) {
    return;
  }

  try {
    window.localStorage?.setItem(THEME_STORAGE_KEY, themeMode);
  } catch {
    // Ignore storage failures so theme changes still update Svelte state and DOM classes.
  }
}

function createThemeStore() {
  const initialThemeMode = getStoredThemeMode() ?? DEFAULT_THEME_MODE;
  const initialResolvedTheme = resolveThemeMode(initialThemeMode);
  const store = writable<ThemeMode>(initialThemeMode);
  const resolvedStore = writable<ResolvedTheme>(initialResolvedTheme);
  let currentThemeMode = initialThemeMode;

  applyResolvedTheme(initialResolvedTheme);
  persistThemeMode(initialThemeMode);

  function applyThemeMode(themeMode: ThemeMode) {
    currentThemeMode = themeMode;
    const nextResolvedTheme = resolveThemeMode(themeMode);

    store.set(themeMode);
    resolvedStore.set(nextResolvedTheme);
    applyResolvedTheme(nextResolvedTheme);
    persistThemeMode(themeMode);
  }

  if (canUseBrowserApis()) {
    const darkSchemeQuery = window.matchMedia?.('(prefers-color-scheme: dark)');
    darkSchemeQuery?.addEventListener('change', (event) => {
      if (currentThemeMode !== 'system') {
        return;
      }

      const nextResolvedTheme: ResolvedTheme = event.matches ? 'dark' : 'light';
      resolvedStore.set(nextResolvedTheme);
      applyResolvedTheme(nextResolvedTheme);
    });
  }

  return {
    subscribe: store.subscribe,
    resolved: resolvedStore,
    setTheme(themeMode: ThemeMode) {
      applyThemeMode(themeMode);
    },
    toggleTheme() {
      applyThemeMode(resolveThemeMode(currentThemeMode) === 'dark' ? 'light' : 'dark');
    }
  };
}

export const theme = createThemeStore();
export const resolvedTheme = theme.resolved;
export const setTheme = theme.setTheme;
export const toggleTheme = theme.toggleTheme;
