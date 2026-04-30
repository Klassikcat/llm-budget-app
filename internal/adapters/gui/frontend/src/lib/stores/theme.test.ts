import { get } from 'svelte/store';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const THEME_STORAGE_KEY = 'llm-budget-tracker-theme';

async function loadThemeStore() {
  vi.resetModules();
  return import('./theme');
}

const mediaListeners: ((event: MediaQueryListEvent) => void)[] = [];

function mockPreferredScheme(scheme: 'dark' | 'light' | null) {
  mediaListeners.length = 0;

  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    writable: true,
    value: vi.fn((query: string) => ({
      matches: scheme ? query === `(prefers-color-scheme: ${scheme})` : false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn((_eventName: string, listener: (event: MediaQueryListEvent) => void) => {
        mediaListeners.push(listener);
      }),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn()
    }))
  });
}

function dispatchSystemThemeChange(matchesDark: boolean) {
  const event = new Event('change') as MediaQueryListEvent;
  Object.defineProperty(event, 'matches', { value: matchesDark });
  mediaListeners.forEach(listener => listener(event));
}

function expectHtmlTheme(expectedTheme: 'dark' | 'light') {
  const oppositeTheme = expectedTheme === 'dark' ? 'light' : 'dark';

  expect(document.documentElement.classList.contains(expectedTheme)).toBe(true);
  expect(document.documentElement.classList.contains(oppositeTheme)).toBe(false);
}

describe('theme store', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.className = '';
    mockPreferredScheme(null);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it('initializes from a valid saved localStorage value', async () => {
    localStorage.setItem(THEME_STORAGE_KEY, 'light');

    const { theme } = await loadThemeStore();

    expect(get(theme)).toBe('light');
    expectHtmlTheme('light');
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('light');
  });

  it('defaults to system mode and resolves OS dark preference when saved theme is missing', async () => {
    mockPreferredScheme('dark');

    const { resolvedTheme, theme } = await loadThemeStore();

    expect(get(theme)).toBe('system');
    expect(get(resolvedTheme)).toBe('dark');
    expectHtmlTheme('dark');
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('system');
  });

  it('falls back to system mode when saved theme is invalid', async () => {
    localStorage.setItem(THEME_STORAGE_KEY, 'solarized');
    mockPreferredScheme('light');

    const { resolvedTheme, theme } = await loadThemeStore();

    expect(get(theme)).toBe('system');
    expect(get(resolvedTheme)).toBe('light');
    expectHtmlTheme('light');
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('system');
  });

  it('uses dark resolved default when storage and OS preference do not provide a theme', async () => {
    const { resolvedTheme, theme } = await loadThemeStore();

    expect(get(theme)).toBe('system');
    expect(get(resolvedTheme)).toBe('dark');
    expectHtmlTheme('dark');
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('system');
  });

  it('adapts to system theme changes while system mode is selected', async () => {
    mockPreferredScheme('dark');

    const { resolvedTheme, setTheme, theme } = await loadThemeStore();

    dispatchSystemThemeChange(false);

    expect(get(theme)).toBe('system');
    expect(get(resolvedTheme)).toBe('light');
    expectHtmlTheme('light');
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('system');

    setTheme('dark');
    dispatchSystemThemeChange(false);

    expect(get(theme)).toBe('dark');
    expect(get(resolvedTheme)).toBe('dark');
    expectHtmlTheme('dark');
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('dark');
  });

  it('setTheme updates mode, resolved theme, html classes, and localStorage', async () => {
    const { resolvedTheme, setTheme, theme } = await loadThemeStore();

    setTheme('light');

    expect(get(theme)).toBe('light');
    expect(get(resolvedTheme)).toBe('light');
    expectHtmlTheme('light');
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('light');
  });

  it('toggleTheme switches between explicit dark and light while updating html classes and localStorage', async () => {
    const { resolvedTheme, theme, toggleTheme } = await loadThemeStore();

    toggleTheme();

    expect(get(theme)).toBe('light');
    expect(get(resolvedTheme)).toBe('light');
    expectHtmlTheme('light');
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('light');

    toggleTheme();

    expect(get(theme)).toBe('dark');
    expect(get(resolvedTheme)).toBe('dark');
    expectHtmlTheme('dark');
    expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe('dark');
  });

  it('does not crash when browser globals are unavailable during module initialization', async () => {
    vi.stubGlobal('window', undefined);
    vi.stubGlobal('document', undefined);

    const { theme, setTheme, toggleTheme } = await loadThemeStore();

    expect(get(theme)).toBe('system');
    expect(() => setTheme('light')).not.toThrow();
    expect(get(theme)).toBe('light');
    expect(() => toggleTheme()).not.toThrow();
    expect(get(theme)).toBe('dark');
  });
});
