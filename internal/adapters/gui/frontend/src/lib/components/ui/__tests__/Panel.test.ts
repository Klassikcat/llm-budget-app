import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import Panel from '../Panel.svelte';

describe('Panel', () => {
  it('renders title when provided', () => {
    const { getByText } = render(Panel, { title: 'Test Panel' });
    expect(getByText('Test Panel')).toBeInTheDocument();
  });

  it('applies custom classes', () => {
    const { container } = render(Panel, { class: 'custom-class' });
    expect(container.firstChild).toHaveClass('custom-class');
  });
});
