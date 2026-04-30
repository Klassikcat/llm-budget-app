<script module lang="ts">
  import type { Component } from 'svelte';

  export interface Column<T = Record<string, unknown>> {
    key: string;
    label: string;
    sortable?: boolean;
    align?: 'left' | 'center' | 'right';
    component?: Component<Record<string, unknown>>;
    componentProps?: (row: T) => Record<string, unknown>;
    format?: (value: unknown, row: T) => string;
  }
</script>

<script lang="ts">
  type Row = $$Generic<Record<string, unknown>>;

  export let data: Row[] = [];
  export let columns: Column<Row>[] = [];
  export let loading: boolean = false;
  export let emptyMessage: string = 'No data available';
  export let sortKey: string | null = null;
  export let sortDirection: 'asc' | 'desc' = 'asc';
  export let onRowClick: ((row: Row) => void) | null = null;
  export let onSort: ((detail: { key: string; direction: 'asc' | 'desc' }) => void) | null = null;

  function handleSort(column: Column<Row>) {
    if (!column.sortable) return;
    
    let newDirection: 'asc' | 'desc' = 'asc';
    if (sortKey === column.key) {
      newDirection = sortDirection === 'asc' ? 'desc' : 'asc';
    }
    
    sortKey = column.key;
    sortDirection = newDirection;
    
    if (onSort) {
      onSort({ key: sortKey, direction: sortDirection });
    }
  }

  function handleRowClick(row: Row) {
    if (onRowClick) {
      onRowClick(row);
    }
  }
</script>

<div class="w-full overflow-x-auto border border-panel-border rounded-compact bg-card">
  <table class="w-full text-left border-collapse">
    <thead>
      <tr class="border-b border-panel-border bg-background-hover">
        {#each columns as column (column.key)}
          <th 
            class="px-md py-sm text-xs font-semibold text-text-muted whitespace-nowrap select-none"
            class:cursor-pointer={column.sortable}
            class:hover:text-text={column.sortable}
            class:text-left={!column.align || column.align === 'left'}
            class:text-center={column.align === 'center'}
            class:text-right={column.align === 'right'}
            on:click={() => handleSort(column)}
          >
            <div class="flex items-center gap-xs" class:justify-end={column.align === 'right'} class:justify-center={column.align === 'center'}>
              {column.label}
              {#if column.sortable}
                <span class="inline-flex flex-col w-3 h-3 opacity-50" class:opacity-100={sortKey === column.key}>
                  {#if sortKey !== column.key || sortDirection === 'asc'}
                    <svg class="w-3 h-3 -mb-1" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="18 15 12 9 6 15"></polyline></svg>
                  {/if}
                  {#if sortKey !== column.key || sortDirection === 'desc'}
                    <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="6 9 12 15 18 9"></polyline></svg>
                  {/if}
                </span>
              {/if}
            </div>
          </th>
        {/each}
      </tr>
    </thead>
    <tbody>
      {#if loading}
        {#each Array(5) as _}
          <tr class="border-b border-panel-border last:border-0">
            {#each columns as column}
              <td class="px-md py-sm">
                <div class="h-4 bg-background-active rounded animate-pulse w-3/4"></div>
              </td>
            {/each}
          </tr>
        {/each}
      {:else if data.length === 0}
        <tr>
          <td colspan={columns.length} class="px-md py-xl text-center text-sm text-text-muted">
            {emptyMessage}
          </td>
        </tr>
      {:else}
        {#each data as row, i}
          <tr 
            class="border-b border-panel-border last:border-0 hover:bg-background-hover transition-colors"
            class:bg-background={i % 2 === 0}
            class:bg-card={i % 2 !== 0}
            class:cursor-pointer={!!onRowClick}
            on:click={() => handleRowClick(row)}
            role={onRowClick ? 'button' : undefined}
            tabindex={onRowClick ? 0 : undefined}
            on:keydown={(e) => e.key === 'Enter' && handleRowClick(row)}
          >
            {#each columns as column (column.key)}
              <td 
                class="px-md py-sm text-sm text-text"
                class:text-left={!column.align || column.align === 'left'}
                class:text-center={column.align === 'center'}
                class:text-right={column.align === 'right'}
              >
                {#if column.component}
                  <svelte:component this={column.component} {...(column.componentProps ? column.componentProps(row) : { value: row[column.key] })} />
                {:else if column.format}
                  {column.format(row[column.key], row)}
                {:else}
                  {row[column.key] ?? ''}
                {/if}
              </td>
            {/each}
          </tr>
        {/each}
      {/if}
    </tbody>
  </table>
</div>
