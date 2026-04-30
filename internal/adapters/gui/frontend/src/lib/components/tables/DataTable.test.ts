import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import DataTable from './DataTable.svelte';

describe('DataTable', () => {
  const columns = [
    { key: 'id', label: 'ID', sortable: true },
    { key: 'name', label: 'Name' },
    { key: 'amount', label: 'Amount', align: 'right' as const }
  ];

  const data = [
    { id: 1, name: 'Item 1', amount: 100 },
    { id: 2, name: 'Item 2', amount: 200 }
  ];

  it('renders headers correctly', () => {
    const { getByText } = render(DataTable, { columns, data });
    expect(getByText('ID')).toBeInTheDocument();
    expect(getByText('Name')).toBeInTheDocument();
    expect(getByText('Amount')).toBeInTheDocument();
  });

  it('renders data rows correctly', () => {
    const { getByText } = render(DataTable, { columns, data });
    expect(getByText('Item 1')).toBeInTheDocument();
    expect(getByText('200')).toBeInTheDocument();
  });

  it('renders empty state when no data', () => {
    const { getByText } = render(DataTable, { columns, data: [], emptyMessage: 'No items found' });
    expect(getByText('No items found')).toBeInTheDocument();
  });

  it('calls onSort when clicking sortable header', async () => {
    const mockSort = vi.fn();
    const { getByText } = render(DataTable, { columns, data, onSort: mockSort });
    
    await fireEvent.click(getByText('ID'));
    
    expect(mockSort).toHaveBeenCalledWith(expect.objectContaining({
      key: 'id', direction: 'asc'
    }));
  });

  it('calls onRowClick when clicking a row', async () => {
    const mockRowClick = vi.fn();
    const { getByText } = render(DataTable, { columns, data, onRowClick: mockRowClick });
    
    await fireEvent.click(getByText('Item 1'));
    
    expect(mockRowClick).toHaveBeenCalledWith(data[0]);
  });
});
