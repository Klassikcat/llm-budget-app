import { describe, it, expect } from 'vitest';
import tailwindConfig from '../../../tailwind.config';

describe('Tailwind Config', () => {
  it('should use class-based dark mode', () => {
    expect(tailwindConfig.darkMode).toBe('class');
  });

  it('should extend colors with CSS variables', () => {
    const colors = tailwindConfig.theme?.extend?.colors as Record<string, string>;
    expect(colors).toBeDefined();
    expect(colors.background).toBe('var(--color-background)');
    expect(colors['background-hover']).toBe('var(--color-background-hover)');
    expect(colors['background-active']).toBe('var(--color-background-active)');
    expect(colors.card).toBe('var(--color-card)');
    expect(colors['card-hover']).toBe('var(--color-card-hover)');
    expect(colors.text).toBe('var(--color-text)');
    expect(colors['text-muted']).toBe('var(--color-text-muted)');
    expect(colors.primary).toBe('var(--color-primary)');
    expect(colors.success).toBe('var(--color-success)');
    expect(colors.warning).toBe('var(--color-warning)');
    expect(colors.danger).toBe('var(--color-danger)');
    expect(colors.muted).toBe('var(--color-muted)');
    expect(colors.border).toBe('var(--color-border)');
    expect(colors['border-hover']).toBe('var(--color-border-hover)');
    expect(colors['panel-border']).toBe('var(--color-panel-border)');
    expect(colors['status-normal']).toBe('var(--color-status-normal)');
  });

  it('should extend spacing with CSS variables', () => {
    const spacing = tailwindConfig.theme?.extend?.spacing as Record<string, string>;
    expect(spacing).toBeDefined();
    expect(spacing.xs).toBe('var(--spacing-xs)');
    expect(spacing.sm).toBe('var(--spacing-sm)');
    expect(spacing.md).toBe('var(--spacing-md)');
    expect(spacing.lg).toBe('var(--spacing-lg)');
    expect(spacing.xl).toBe('var(--spacing-xl)');
    expect(spacing['2xl']).toBe('var(--spacing-2xl)');
    expect(spacing['panel-padding']).toBe('var(--spacing-panel-padding)');
    expect(spacing['grid-gap']).toBe('var(--spacing-grid-gap)');
    expect(spacing['metric-sizing']).toBe('var(--spacing-metric-sizing)');
  });

  it('should define typography tokens', () => {
    const fontFamily = tailwindConfig.theme?.extend?.fontFamily as Record<string, string[]>;
    expect(fontFamily).toBeDefined();
    expect(fontFamily.sans).toContain('Inter');
    expect(fontFamily.mono).toContain('"JetBrains Mono"');

    const fontSize = tailwindConfig.theme?.extend?.fontSize as Record<string, string>;
    expect(fontSize).toBeDefined();
    expect(fontSize.xs).toBe('var(--font-size-xs)');
    expect(fontSize.sm).toBe('var(--font-size-sm)');
    expect(fontSize.base).toBe('var(--font-size-base)');
    expect(fontSize.lg).toBe('var(--font-size-lg)');
    expect(fontSize.xl).toBe('var(--font-size-xl)');

    const fontWeight = tailwindConfig.theme?.extend?.fontWeight as Record<string, string>;
    expect(fontWeight).toBeDefined();
    expect(fontWeight.normal).toBe('var(--font-weight-normal)');
    expect(fontWeight.medium).toBe('var(--font-weight-medium)');
    expect(fontWeight.semibold).toBe('var(--font-weight-semibold)');
    expect(fontWeight.bold).toBe('var(--font-weight-bold)');
  });

  it('should define border radius and shadow tokens', () => {
    const borderRadius = tailwindConfig.theme?.extend?.borderRadius as Record<string, string>;
    expect(borderRadius).toBeDefined();
    expect(borderRadius.compact).toBe('var(--radius-compact)');

    const boxShadow = tailwindConfig.theme?.extend?.boxShadow as Record<string, string>;
    expect(boxShadow).toBeDefined();
    expect(boxShadow.glow).toBe('var(--shadow-glow)');
  });
});
