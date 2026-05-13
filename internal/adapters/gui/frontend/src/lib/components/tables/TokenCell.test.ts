import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import TokenCell from './TokenCell.svelte';

describe('TokenCell', () => {
  it('formats small numbers normally', () => {
    const { getByText } = render(TokenCell, { value: 500 });
    expect(getByText('500')).toBeInTheDocument();
  });

  it('formats thousands with K', () => {
    const { getByText } = render(TokenCell, { value: 1200 });
    expect(getByText('1.2K')).toBeInTheDocument();
  });

  it('formats millions with M', () => {
    const { getByText } = render(TokenCell, { value: 1500000 });
    expect(getByText('1.5M')).toBeInTheDocument();
  });
});
