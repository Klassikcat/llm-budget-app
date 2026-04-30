import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import SparklineCard from '../SparklineCard.svelte';

describe('SparklineCard', () => {
  it('renders label and value', () => {
    const { getByText } = render(SparklineCard, { label: 'Active Users', value: '1,234' });
    expect(getByText('Active Users')).toBeInTheDocument();
    expect(getByText('1,234')).toBeInTheDocument();
  });

  it('renders empty state when no data', () => {
    const { getByText } = render(SparklineCard, { label: 'Active Users', value: '1,234', data: [] });
    expect(getByText('No data')).toBeInTheDocument();
  });

  it('renders svg path when data is provided', () => {
    const { container } = render(SparklineCard, { label: 'Active Users', value: '1,234', data: [10, 20, 15, 30] });
    const path = container.querySelector('path');
    expect(path).toBeInTheDocument();
    expect(path?.getAttribute('d')).toContain('M');
  });
});
