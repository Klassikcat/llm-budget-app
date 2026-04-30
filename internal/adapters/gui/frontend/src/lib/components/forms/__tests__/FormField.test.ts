import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import FormField from '../FormField.svelte';

describe('FormField', () => {
  it('renders label and help text', () => {
    const { getByText } = render(FormField, { label: 'Test Label', helpText: 'Some help text' });
    expect(getByText('Test Label')).toBeInTheDocument();
    expect(getByText('Some help text')).toBeInTheDocument();
  });

  it('renders error text instead of help text when error is provided', () => {
    const { getByText, queryByText } = render(FormField, { 
      label: 'Test Label', 
      helpText: 'Some help text',
      error: 'This field is required'
    });
    
    expect(getByText('This field is required')).toBeInTheDocument();
    expect(getByText('This field is required')).toHaveClass('text-danger');
    expect(queryByText('Some help text')).not.toBeInTheDocument();
  });

  it('shows required asterisk when required is true', () => {
    const { getByText } = render(FormField, { label: 'Test Label', required: true });
    const asterisk = getByText('*');
    expect(asterisk).toBeInTheDocument();
    expect(asterisk).toHaveClass('text-danger');
  });
});
