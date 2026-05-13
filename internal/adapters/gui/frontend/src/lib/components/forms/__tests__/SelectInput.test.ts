import { describe, it, expect } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import SelectInput from '../SelectInput.svelte';

describe('SelectInput', () => {
  const options = [
    { value: '1', label: 'Option 1' },
    { value: '2', label: 'Option 2' },
    { value: '3', label: 'Option 3', disabled: true }
  ];

  it('renders correctly with options', () => {
    const { getByRole, getAllByRole } = render(SelectInput, { options, id: 'test-select', value: '' });
    const select = getByRole('combobox');
    expect(select).toBeInTheDocument();
    
    const optionElements = getAllByRole('option', { hidden: true });
    // +1 for the empty hidden option when not required and no value
    expect(optionElements.length).toBe(4);
    expect(optionElements[1]).toHaveTextContent('Option 1');
    expect(optionElements[3]).toBeDisabled();
  });

  it('binds value correctly', async () => {
    const { getByRole } = render(SelectInput, { options, value: '1' });
    const select = getByRole('combobox') as HTMLSelectElement;
    expect(select.value).toBe('1');

    await fireEvent.change(select, { target: { value: '2' } });
    expect(select.value).toBe('2');
  });
});
