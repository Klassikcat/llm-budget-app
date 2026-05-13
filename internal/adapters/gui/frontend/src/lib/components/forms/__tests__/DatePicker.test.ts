import { describe, it, expect } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import DatePicker from '../DatePicker.svelte';

describe('DatePicker', () => {
  it('renders correctly', () => {
    render(DatePicker, { id: 'test-date' });
    // In jsdom, type="date" might not have a specific role, but we can query by id or tag
    const input = document.getElementById('test-date') as HTMLInputElement;
    expect(input).toBeInTheDocument();
    expect(input.type).toBe('date');
  });

  it('binds value correctly', async () => {
    render(DatePicker, { id: 'test-date', value: '2023-01-01' });
    const input = document.getElementById('test-date') as HTMLInputElement;
    expect(input.value).toBe('2023-01-01');

    await fireEvent.input(input, { target: { value: '2023-12-31' } });
    expect(input.value).toBe('2023-12-31');
  });
});
