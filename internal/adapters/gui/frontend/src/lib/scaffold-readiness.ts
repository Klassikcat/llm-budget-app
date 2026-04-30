export const appTitle = 'LLM Budget Tracker';

export const readinessItems = [
  'SvelteKit static SPA scaffold',
  'Wails asset output: dist',
  'TypeScript and Vitest enabled'
] as const;

export function hasReadinessItem(item: string): boolean {
  return readinessItems.includes(item as (typeof readinessItems)[number]);
}
