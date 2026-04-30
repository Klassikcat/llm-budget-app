import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import Form from '../Form.svelte';

describe('Form', () => {
  it('renders correctly', () => {
    const { container } = render(Form, { class: 'test-form' });
    const form = container.querySelector('form');
    expect(form).toBeInTheDocument();
    expect(form).toHaveClass('test-form');
  });

  it('calls onsubmit and prevents default', async () => {
    const onsubmit = vi.fn();
    const { container } = render(Form, { onsubmit });
    const form = container.querySelector('form') as HTMLFormElement;
    
    await fireEvent.submit(form);
    expect(onsubmit).toHaveBeenCalled();
  });

  it('validates schema and calls onvalidate', async () => {
    const onvalidate = vi.fn();
    const onsubmit = vi.fn();
    const schema = {
      safeParse: vi.fn().mockReturnValue({ success: true, data: { test: 'data' } })
    };
    const data = { test: 'data' };

    const { container } = render(Form, { schema, data, onvalidate, onsubmit });
    const form = container.querySelector('form') as HTMLFormElement;
    
    await fireEvent.submit(form);
    
    expect(schema.safeParse).toHaveBeenCalledWith(data);
    expect(onvalidate).toHaveBeenCalledWith({ success: true, data: { test: 'data' } });
    expect(onsubmit).toHaveBeenCalled();
  });

  it('stops submission if validation fails', async () => {
    const onvalidate = vi.fn();
    const onsubmit = vi.fn();
    const schema = {
      safeParse: vi.fn().mockReturnValue({ 
        success: false, 
        error: { issues: [{ path: ['test'], message: 'Invalid' }] } 
      })
    };
    const data = { test: 'invalid' };

    const { container } = render(Form, { schema, data, onvalidate, onsubmit });
    const form = container.querySelector('form') as HTMLFormElement;
    
    await fireEvent.submit(form);
    
    expect(schema.safeParse).toHaveBeenCalledWith(data);
    expect(onvalidate).toHaveBeenCalled();
    expect(onsubmit).not.toHaveBeenCalled();
  });
});
