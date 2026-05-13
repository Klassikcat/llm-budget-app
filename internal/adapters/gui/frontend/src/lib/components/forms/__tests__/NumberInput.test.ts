import { describe, it, expect } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import NumberInput from '../NumberInput.svelte';

describe('NumberInput', () => {
  it('renders correctly', () => {
    const { getByRole } = render(NumberInput, { id: 'test-number', placeholder: 'Enter number' });
    const input = getByRole('spinbutton');
    expect(input).toBeInTheDocument();
    expect(input).toHaveAttribute('type', 'number');
    expect(input).toHaveAttribute('id', 'test-number');
    expect(input).toHaveAttribute('placeholder', 'Enter number');
  });

  it('binds value correctly', async () => {
    const { getByRole } = render(NumberInput, { value: 10 });
    const input = getByRole('spinbutton') as HTMLInputElement;
    expect(input.value).toBe('10');

    await fireEvent.input(input, { target: { value: '20' } });
    expect(input.value).toBe('20');
  });

  it('respects min and max constraints', () => {
    const { getByRole } = render(NumberInput, { min: 0, max: 100 });
    const input = getByRole('spinbutton');
    expect(input).toHaveAttribute('min', '0');
    expect(input).toHaveAttribute('max', '100');
  });
});
