import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import DateCell from './DateCell.svelte';

describe('DateCell', () => {
  it('formats short dates correctly', () => {
    const date = new Date('2024-01-15T10:30:00Z');
    const { getByText } = render(DateCell, { value: date, format: 'short' });
    expect(getByText(/2024/)).toBeInTheDocument();
  });

  it('formats long dates correctly', () => {
    const date = new Date('2024-01-15T10:30:00Z');
    const { getByText } = render(DateCell, { value: date, format: 'long' });
    expect(getByText(/2024/)).toBeInTheDocument();
    expect(getByText(/:/)).toBeInTheDocument();
  });

  it('handles invalid dates gracefully', () => {
    const { getByText } = render(DateCell, { value: 'invalid-date' });
    expect(getByText('Invalid Date')).toBeInTheDocument();
  });
});
