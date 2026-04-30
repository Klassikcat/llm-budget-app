import { afterEach, describe, expect, it } from 'vitest';
import { createWailsBindingClient, getGeneratedBinding } from './client';

const originalGo = window.go;

afterEach(() => {
  if (originalGo === undefined) {
    delete window.go;
    return;
  }
  window.go = originalGo;
});

describe('getGeneratedBinding', () => {
  it('reads Wails bindings from the injected window.go namespace', () => {
    const loadGraphs = async (...args: unknown[]) => ({ month: args[0] });
    window.go = {
      gui: {
        GraphsBinding: {
          LoadGraphs: loadGraphs
        }
      }
    };

    expect(getGeneratedBinding('GraphsBinding').LoadGraphs).toBe(loadGraphs);
  });

  it('reports unavailable bindings without attempting module imports', () => {
    window.go = { gui: {} };

    expect(() => getGeneratedBinding('FormsBinding')).toThrow('Wails module FormsBinding did not export a method map');
  });
});

describe('createWailsBindingClient', () => {
  it('invokes methods from the injected Wails binding namespace', async () => {
    window.go = {
      gui: {
        DashboardBinding: {
          LoadDashboard: async (...args: unknown[]) => ({ month: args[0], empty: false })
        }
      }
    };

    await expect(createWailsBindingClient().invoke('DashboardBinding', 'LoadDashboard', '2026-04')).resolves.toEqual({
      month: '2026-04',
      empty: false
    });
  });
});
