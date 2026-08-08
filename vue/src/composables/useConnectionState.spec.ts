import { describe, it, expect, beforeEach } from 'vitest';
import { useConnectionState } from '@/composables/useConnectionState';
import {
  createPollingTransport,
  setLiveTransport,
  startLive,
  stopLive,
  type ConnectionState,
  type LiveTransport,
} from '@/plugins/live';
import { ref, readonly, type Ref } from 'vue';

/** A transport whose state a test can drive directly. */
function fakeTransport(): LiveTransport & { set: (s: ConnectionState) => void } {
  const state = ref<ConnectionState>('offline');
  return {
    name: 'fake',
    state: readonly(state) as Readonly<Ref<ConnectionState>>,
    start: () => {
      state.value = 'live';
    },
    stop: () => {
      state.value = 'offline';
    },
    set: (s: ConnectionState) => {
      state.value = s;
    },
  };
}

beforeEach(() => {
  stopLive();
  setLiveTransport(createPollingTransport());
});

describe('useConnectionState', () => {
  it('reports polling — never live — while the polling transport is running', () => {
    startLive('2026');

    const { state, label, isHealthy, isDisconnected } = useConnectionState();

    // Claiming "live" for an interval-based transport would be exactly the
    // dishonesty the indicator exists to prevent.
    expect(state.value).toBe('polling');
    expect(label.value).toBe('Opdaterer periodisk');
    expect(isHealthy.value).toBe(true);
    expect(isDisconnected.value).toBe(false);
  });

  it('is offline before starting and after stopping', () => {
    const { state, isDisconnected } = useConnectionState();
    expect(state.value).toBe('offline');

    startLive('2026');
    expect(state.value).toBe('polling');

    stopLive();
    expect(state.value).toBe('offline');
    expect(isDisconnected.value).toBe(true);
  });

  it('follows whichever transport is installed', () => {
    const fake = fakeTransport();
    setLiveTransport(fake);

    const { state, label } = useConnectionState();

    fake.set('live');
    expect(state.value).toBe('live');
    expect(label.value).toBe('Live');

    fake.set('reconnecting');
    expect(state.value).toBe('reconnecting');
    expect(label.value).toBe('Forbinder igen…');
  });

  it('gives every state a Danish label and description', () => {
    const fake = fakeTransport();
    setLiveTransport(fake);
    const { label, description } = useConnectionState();

    for (const s of ['live', 'reconnecting', 'polling', 'offline'] as ConnectionState[]) {
      fake.set(s);
      expect(label.value.length).toBeGreaterThan(0);
      expect(description.value.length).toBeGreaterThan(0);
    }
  });

  it('treats only reconnecting and offline as disconnected', () => {
    const fake = fakeTransport();
    setLiveTransport(fake);
    const { isDisconnected, isHealthy } = useConnectionState();

    fake.set('live');
    expect(isDisconnected.value).toBe(false);
    fake.set('polling');
    expect(isDisconnected.value).toBe(false);
    fake.set('reconnecting');
    expect(isDisconnected.value).toBe(true);
    expect(isHealthy.value).toBe(false);
    fake.set('offline');
    expect(isDisconnected.value).toBe(true);
  });
});
