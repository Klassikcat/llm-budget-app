import { describe, it, expect } from 'vitest';
import fs from 'fs';
import path from 'path';

describe('CSS Tokens', () => {
  const tokensPath = path.resolve(__dirname, './tokens.css');
  const tokensContent = fs.readFileSync(tokensPath, 'utf-8');

  it('should define light theme background', () => {
    expect(tokensContent).toMatch(/--color-background:\s*#f5f5f5;/);
  });

  it('should define dark theme background', () => {
    expect(tokensContent).toMatch(/--color-background:\s*#0f1117;/);
  });

  it('should define primary color tokens', () => {
    expect(tokensContent).toMatch(/--color-primary:/);
    expect(tokensContent).toMatch(/--color-success:/);
    expect(tokensContent).toMatch(/--color-warning:/);
    expect(tokensContent).toMatch(/--color-danger:/);
    expect(tokensContent).toMatch(/--color-muted:/);
  });

  it('should define spacing tokens', () => {
    expect(tokensContent).toMatch(/--spacing-xs:\s*2px;/);
    expect(tokensContent).toMatch(/--spacing-sm:\s*4px;/);
    expect(tokensContent).toMatch(/--spacing-md:\s*8px;/);
    expect(tokensContent).toMatch(/--spacing-lg:\s*16px;/);
    expect(tokensContent).toMatch(/--spacing-xl:\s*24px;/);
    expect(tokensContent).toMatch(/--spacing-2xl:\s*32px;/);
  });

  it('should define dense dashboard tokens', () => {
    expect(tokensContent).toMatch(/--spacing-panel-padding:\s*12px;/);
    expect(tokensContent).toMatch(/--spacing-grid-gap:\s*8px;/);
    expect(tokensContent).toMatch(/--spacing-metric-sizing:\s*24px;/);
    expect(tokensContent).toMatch(/--radius-compact:\s*4px;/);
    expect(tokensContent).toMatch(/--shadow-glow:/);
  });

  it('should define status tokens', () => {
    expect(tokensContent).toMatch(/--color-status-normal:/);
    expect(tokensContent).toMatch(/--color-status-success:/);
    expect(tokensContent).toMatch(/--color-status-warning:/);
    expect(tokensContent).toMatch(/--color-status-danger:/);
    expect(tokensContent).toMatch(/--color-status-inactive:/);
  });

  it('should define typography tokens', () => {
    expect(tokensContent).toMatch(/--font-size-xs:/);
    expect(tokensContent).toMatch(/--font-size-sm:/);
    expect(tokensContent).toMatch(/--font-size-base:/);
    expect(tokensContent).toMatch(/--font-size-lg:/);
    expect(tokensContent).toMatch(/--font-size-xl:/);
    expect(tokensContent).toMatch(/--font-weight-normal:/);
    expect(tokensContent).toMatch(/--font-weight-medium:/);
    expect(tokensContent).toMatch(/--font-weight-semibold:/);
    expect(tokensContent).toMatch(/--font-weight-bold:/);
  });

  it('should support dark and light theme selectors with dark as default', () => {
    expect(tokensContent).toMatch(/:root,\s*\.dark,\s*html\.dark\s*\{/);
    expect(tokensContent).toMatch(/\.light,\s*html\.light\s*\{/);
  });
});
