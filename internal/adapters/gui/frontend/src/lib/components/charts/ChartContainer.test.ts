import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import ChartContainer from './ChartContainer.svelte';

describe('ChartContainer', () => {
  it('renders title when provided', () => {
    const { getByText } = render(ChartContainer, { title: 'Test Chart' });
    expect(getByText('Test Chart')).toBeInTheDocument();
  });

  it('renders loading state', () => {
    const { getByText } = render(ChartContainer, { loading: true });
    expect(getByText('Loading...')).toBeInTheDocument();
  });

  it('renders error state', () => {
    const { getByText } = render(ChartContainer, { error: 'Failed to load' });
    expect(getByText('Failed to load')).toBeInTheDocument();
  });

  it('renders empty state with exact text', () => {
    const { getByText } = render(ChartContainer, { empty: true });
    expect(getByText('No data available')).toBeInTheDocument();
  });
});
