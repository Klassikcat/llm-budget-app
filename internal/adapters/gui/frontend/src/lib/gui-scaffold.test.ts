import { describe, expect, it } from 'vitest';

import { appTitle, hasReadinessItem, readinessItems } from './scaffold-readiness';

describe('GUI scaffold readiness metadata', () => {
  it('identifies the app and confirms the Wails dist output contract', () => {
    expect(appTitle).toBe('LLM Budget Tracker');
    expect(readinessItems).toHaveLength(3);
    expect(hasReadinessItem('Wails asset output: dist')).toBe(true);
    expect(hasReadinessItem('Generated bindings consumed')).toBe(false);
  });
});
