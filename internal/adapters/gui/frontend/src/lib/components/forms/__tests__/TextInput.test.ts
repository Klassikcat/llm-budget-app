import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import TextInput from '../TextInput.svelte';

describe('TextInput', () => {
  it('renders correctly', () => {
    const { getByRole } = render(TextInput, { id: 'test-input', placeholder: 'Enter text' });
    const input = getByRole('textbox');
    expect(input).toBeInTheDocument();
    expect(input).toHaveAttribute('id', 'test-input');
    expect(input).toHaveAttribute('placeholder', 'Enter text');
  });

  it('binds value correctly', async () => {
    const { getByRole } = render(TextInput, { value: 'initial' });
    const input = getByRole('textbox') as HTMLInputElement;
    expect(input.value).toBe('initial');

    await fireEvent.input(input, { target: { value: 'new value' } });
    expect(input.value).toBe('new value');
  });

  it('applies error styling when error is true', () => {
    const { getByRole } = render(TextInput, { error: true });
    const input = getByRole('textbox');
    expect(input).toHaveClass('border-danger');
  });

  it('calls onchange handler', async () => {
    const onchange = vi.fn();
    const { getByRole } = render(TextInput, { onchange });
    const input = getByRole('textbox');
    
    await fireEvent.change(input, { target: { value: 'changed' } });
    expect(onchange).toHaveBeenCalled();
  });
});
