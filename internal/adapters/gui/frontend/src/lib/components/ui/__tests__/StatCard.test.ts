import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import StatCard from '../StatCard.svelte';

describe('StatCard', () => {
  it('renders label and value', () => {
    const { getByText } = render(StatCard, { label: 'Total Spend', value: '$42.50' });
    expect(getByText('Total Spend')).toBeInTheDocument();
    expect(getByText('$42.50')).toBeInTheDocument();
  });

  it('renders trend up indicator', () => {
    const { container } = render(StatCard, { label: 'Total Spend', value: '$42.50', trend: 'up' });
    const svg = container.querySelector('svg[aria-label="Trend up"]');
    expect(svg).toBeInTheDocument();
  });

  it('renders trend down indicator', () => {
    const { container } = render(StatCard, { label: 'Total Spend', value: '$42.50', trend: 'down' });
    const svg = container.querySelector('svg[aria-label="Trend down"]');
    expect(svg).toBeInTheDocument();
  });

  it('renders trend value', () => {
    const { getByText } = render(StatCard, { label: 'Total Spend', value: '$42.50', trend: 'up', trendValue: '5%' });
    expect(getByText('5%')).toBeInTheDocument();
  });
});
