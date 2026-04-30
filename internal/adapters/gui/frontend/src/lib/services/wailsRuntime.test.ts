import { afterEach, describe, expect, it } from 'vitest';
import { getWailsRuntime } from './wailsRuntime';

const originalRuntime = window.runtime;

afterEach(() => {
  if (originalRuntime === undefined) {
    delete window.runtime;
    return;
  }
  window.runtime = originalRuntime;
});

describe('getWailsRuntime', () => {
  it('reads the Wails runtime from the injected window.runtime namespace', () => {
    const EventsOn = () => undefined;
    window.runtime = { EventsOn };

    expect(getWailsRuntime()?.EventsOn).toBe(EventsOn);
  });

  it('returns null when the Wails runtime is not injected', () => {
    delete window.runtime;

    expect(getWailsRuntime()).toBeNull();
  });
});
