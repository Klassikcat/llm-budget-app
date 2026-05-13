/// <reference types="@testing-library/jest-dom" />
import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/svelte';
import Header from './Header.svelte';

vi.mock('$app/stores', () => {
  return {
    page: {
      subscribe: vi.fn((fn) => {
        fn({ url: { pathname: '/usage' } });
        return () => {};
      })
    }
  };
});

describe('Header Component', () => {
  it('renders the correct page title based on the route', () => {
    const { getByText } = render(Header);
    expect(getByText('Usage')).toBeInTheDocument();
  });

  it('renders the refresh button', () => {
    const { getByLabelText } = render(Header);
    expect(getByLabelText('Refresh data')).toBeInTheDocument();
  });
});
