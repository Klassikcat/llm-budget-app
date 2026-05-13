/// <reference types="@testing-library/jest-dom" />
import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import Sidebar from './Sidebar.svelte';

vi.mock('$app/stores', () => {
  return {
    page: {
      subscribe: vi.fn((fn) => {
        fn({ url: { pathname: '/' } });
        return () => {};
      })
    }
  };
});

describe('Sidebar Component', () => {
  it('renders all navigation items', () => {
    const { getByText } = render(Sidebar);
    
    expect(getByText('Dashboard')).toBeInTheDocument();
    expect(getByText('Usage')).toBeInTheDocument();
    expect(getByText('Subscriptions')).toBeInTheDocument();
    expect(getByText('Budgets')).toBeInTheDocument();
    expect(getByText('Insights')).toBeInTheDocument();
    expect(getByText('Graphs')).toBeInTheDocument();
    expect(getByText('Settings')).toBeInTheDocument();
  });

  it('highlights the active route', () => {
    const { container } = render(Sidebar);
    
    const dashboardLink = container.querySelector('a[href="/"]');
    expect(dashboardLink).toHaveAttribute('aria-current', 'page');
    expect(dashboardLink).toHaveClass('bg-background-active');
    
    const usageLink = container.querySelector('a[href="/usage"]');
    expect(usageLink).not.toHaveAttribute('aria-current');
    expect(usageLink).not.toHaveClass('bg-background-active');
  });

  it('toggles collapse state when button is clicked', async () => {
    const { getByLabelText, container } = render(Sidebar);
    
    const aside = container.querySelector('aside');
    expect(aside).toHaveClass('w-60');
    
    const toggleButton = getByLabelText('Collapse sidebar');
    await fireEvent.click(toggleButton);
    
    expect(aside).toHaveClass('w-16');
    expect(getByLabelText('Expand sidebar')).toBeInTheDocument();
  });

  it('renders the theme toggle button', () => {
    const { getByLabelText } = render(Sidebar);
    expect(getByLabelText('Toggle theme')).toBeInTheDocument();
  });
});
