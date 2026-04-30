import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import StatusBadge from './StatusBadge.svelte';

describe('StatusBadge', () => {
  it('renders active status with success colors', () => {
    const { getByText } = render(StatusBadge, { status: 'Active' });
    const badge = getByText('Active');
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveClass('text-status-success');
  });

  it('renders inactive status with inactive colors', () => {
    const { getByText } = render(StatusBadge, { status: 'Inactive' });
    const badge = getByText('Inactive');
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveClass('text-status-inactive');
  });

  it('renders expired status with inactive colors', () => {
    const { getByText } = render(StatusBadge, { status: 'Expired' });
    const badge = getByText('Expired');
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveClass('text-status-inactive');
  });

  it('renders over budget status with danger colors', () => {
    const { getByText } = render(StatusBadge, { status: 'Over Budget' });
    const badge = getByText('Over Budget');
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveClass('text-status-danger');
  });

  it('renders pending status with warning colors', () => {
    const { getByText } = render(StatusBadge, { status: 'Pending' });
    const badge = getByText('Pending');
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveClass('text-status-warning');
  });

  it('renders unknown status with normal colors', () => {
    const { getByText } = render(StatusBadge, { status: 'Unknown' });
    const badge = getByText('Unknown');
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveClass('text-status-normal');
  });
});
