import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import CurrencyCell from './CurrencyCell.svelte';

describe('CurrencyCell', () => {
  it('formats whole numbers correctly', () => {
    const { getByText } = render(CurrencyCell, { value: 20 });
    expect(getByText('$20.00')).toBeInTheDocument();
  });

  it('formats decimal numbers correctly', () => {
    const { getByText } = render(CurrencyCell, { value: 1234.5 });
    expect(getByText('$1,234.50')).toBeInTheDocument();
  });

  it('supports different currencies', () => {
    const { getByText } = render(CurrencyCell, { value: 100, currency: 'EUR' });
    expect(getByText('€100.00')).toBeInTheDocument();
  });

  it('falls back to USD for blank currencies', () => {
    const { getByText } = render(CurrencyCell, { value: 12, currency: '' });
    expect(getByText('$12.00')).toBeInTheDocument();
  });
});
