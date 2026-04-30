import { fireEvent, render, screen } from '@testing-library/svelte';
import { beforeEach, describe, expect, it } from 'vitest';
import NotificationCenter from './NotificationCenter.svelte';
import { notificationStore } from '$lib/stores/notification';

describe('NotificationCenter', () => {
  beforeEach(() => {
    notificationStore.reset();
  });

  it('does not render an empty panel that can affect page layout', () => {
    render(NotificationCenter);

    expect(screen.queryByLabelText('Notifications')).not.toBeInTheDocument();
  });

  it('renders notifications in a fixed overlay with stable viewport width', () => {
    notificationStore.addNotification({
      id: 'n-1',
      title: 'A very long notification title that should not force the page layout to expand horizontally',
      body: 'A very long notification body that should wrap inside the toast instead of breaking the surrounding page layout or overflowing the viewport.',
      kind: 'test',
      severity: 'warning'
    });

    render(NotificationCenter);

    const center = screen.getByLabelText('Notifications');
    expect(center).toHaveClass('fixed');
    expect(center.className).toContain('w-[min(calc(100vw-2rem),28rem)]');
    expect(screen.getByText(/A very long notification title/)).toBeInTheDocument();
    expect(screen.getByText(/A very long notification body/)).toBeInTheDocument();
  });

  it('clears notifications without leaving the center mounted', async () => {
    notificationStore.addNotification({
      id: 'n-1',
      title: 'Saved',
      body: 'Settings saved',
      kind: 'settings',
      severity: 'success'
    });

    render(NotificationCenter);

    await fireEvent.click(screen.getByText('Clear All'));

    expect(screen.queryByLabelText('Notifications')).not.toBeInTheDocument();
  });
});
