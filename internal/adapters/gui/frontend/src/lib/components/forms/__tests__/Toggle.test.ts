import { describe, it, expect } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import Toggle from '../Toggle.svelte';

describe('Toggle', () => {
  it('renders correctly', () => {
    const { getByRole } = render(Toggle, { id: 'test-toggle' });
    const checkbox = getByRole('checkbox');
    expect(checkbox).toBeInTheDocument();
    expect(checkbox).not.toBeChecked();
  });

  it('binds checked state correctly', async () => {
    const { getByRole } = render(Toggle, { checked: false });
    const checkbox = getByRole('checkbox') as HTMLInputElement;
    
    await fireEvent.click(checkbox);
    expect(checkbox).toBeChecked();
  });

  it('respects disabled state', () => {
    const { getByRole } = render(Toggle, { disabled: true });
    const checkbox = getByRole('checkbox');
    expect(checkbox).toBeDisabled();
  });
});
