import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import AlertCard from '../AlertCard.svelte';

describe('AlertCard', () => {
  it('renders title and message', () => {
    const { getByText } = render(AlertCard, { title: 'Warning', message: 'Something went wrong' });
    expect(getByText('Warning')).toBeInTheDocument();
    expect(getByText('Something went wrong')).toBeInTheDocument();
  });

  it('applies variant classes', () => {
    const { container } = render(AlertCard, { title: 'Success', variant: 'success' });
    expect(container.firstChild).toHaveClass('border-status-success');
  });
});
