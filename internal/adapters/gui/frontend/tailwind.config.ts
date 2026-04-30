import type { Config } from 'tailwindcss';

export default {
  content: ['./src/**/*.{html,js,svelte,ts}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        background: 'var(--color-background)',
        'background-hover': 'var(--color-background-hover)',
        'background-active': 'var(--color-background-active)',
        card: 'var(--color-card)',
        'card-hover': 'var(--color-card-hover)',
        text: 'var(--color-text)',
        'text-muted': 'var(--color-text-muted)',
        primary: 'var(--color-primary)',
        success: 'var(--color-success)',
        warning: 'var(--color-warning)',
        danger: 'var(--color-danger)',
        muted: 'var(--color-muted)',
        border: 'var(--color-border)',
        'border-hover': 'var(--color-border-hover)',
        'panel-border': 'var(--color-panel-border)',
        'status-normal': 'var(--color-status-normal)',
        'status-success': 'var(--color-status-success)',
        'status-warning': 'var(--color-status-warning)',
        'status-danger': 'var(--color-status-danger)',
        'status-inactive': 'var(--color-status-inactive)',
      },
      spacing: {
        xs: 'var(--spacing-xs)',
        sm: 'var(--spacing-sm)',
        md: 'var(--spacing-md)',
        lg: 'var(--spacing-lg)',
        xl: 'var(--spacing-xl)',
        '2xl': 'var(--spacing-2xl)',
        'panel-padding': 'var(--spacing-panel-padding)',
        'grid-gap': 'var(--spacing-grid-gap)',
        'metric-sizing': 'var(--spacing-metric-sizing)',
      },
      fontFamily: {
        sans: ['Inter', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        mono: ['"JetBrains Mono"', 'ui-monospace', 'SFMono-Regular', 'monospace'],
      },
      fontSize: {
        xs: 'var(--font-size-xs)',
        sm: 'var(--font-size-sm)',
        base: 'var(--font-size-base)',
        lg: 'var(--font-size-lg)',
        xl: 'var(--font-size-xl)',
      },
      fontWeight: {
        normal: 'var(--font-weight-normal)',
        medium: 'var(--font-weight-medium)',
        semibold: 'var(--font-weight-semibold)',
        bold: 'var(--font-weight-bold)',
      },
      borderRadius: {
        compact: 'var(--radius-compact)',
      },
      boxShadow: {
        glow: 'var(--shadow-glow)',
      },
    },
  },
  plugins: [],
} satisfies Config;
