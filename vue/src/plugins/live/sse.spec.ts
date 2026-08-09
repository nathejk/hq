import { describe, it, expect, beforeEach } from 'vitest';
import bus from '@/plugins/bus';
import { createSseTransport, type EventSourceLike } from '@/plugins/live/sse';
import {
  SIGNAL_ENTITIES,
  SIGNAL_ENTITY_CHANGED,
  SIGNAL_RESYNC,
  knownEntities,
  resetKnownEntities,
  type LiveSignal,
} from '@/plugins/live';

/**
 * Stand-in for EventSource, driven by the test.
 *
 * Unit tests run in a node environment with no EventSource, which is exactly why
 * the transport takes a factory.
 */
class FakeEventSource implements EventSourceLike {
  static last: FakeEventSource | undefined;

  readyState = 0;
  onopen: ((ev: unknown) => void) | null = null;
  onerror: ((ev: unknown) => void) | null = null;
  closed = false;

  private listeners = new Map<string, (ev: { data: string }) => void>();

  constructor(public url: string) {
    FakeEventSource.last = this;
  }

  addEventListener(type: string, listener: (ev: { data: string }) => void) {
    this.listeners.set(type, listener);
  }

  close() {
    this.closed = true;
    this.readyState = 2;
  }

  // --- test drivers ---

  open() {
    this.readyState = 1;
    this.onopen?.({});
  }

  emit(type: string, data: unknown) {
    this.listeners.get(type)?.({ data: typeof data === 'string' ? data : JSON.stringify(data) });
  }

  fail(readyState: number) {
    this.readyState = readyState;
    this.onerror?.({});
  }
}

/** Collect signals published onto the bus. */
function collect() {
  const received: LiveSignal[] = [];
  const handler = (s: LiveSignal) => received.push(s);
  bus.on('live', handler);
  return {
    received,
    stop: () => bus.off('live', handler),
  };
}

const newTransport = (entities?: string[]) =>
  createSseTransport({
    entities,
    create: (url) => new FakeEventSource(url),
  });

beforeEach(() => {
  FakeEventSource.last = undefined;
});

describe('createSseTransport', () => {
  it('is not live until the connection opens', () => {
    const t = newTransport();
    t.start('2026');

    // Claiming live before the server has answered would be exactly the
    // dishonesty the connection indicator exists to prevent.
    expect(t.state.value).toBe('reconnecting');

    FakeEventSource.last!.open();
    expect(t.state.value).toBe('live');
  });

  it('sends the year explicitly, and entities when filtered', () => {
    newTransport(['sos', 'patrulje']).start('2025');

    const url = FakeEventSource.last!.url;
    expect(url).toContain('/api/stream?');
    expect(url).toContain('year=2025');
    expect(url).toContain('entities=sos%2Cpatrulje');
  });

  it('sends an empty year rather than omitting it', () => {
    // Empty means "current year". Relying on the server's default instead would
    // make a mismatch between the stream's year and REST's year invisible.
    newTransport().start('');
    expect(FakeEventSource.last!.url).toContain('year=');
  });

  it('emits a resync on connect and on every reconnect', () => {
    const { received, stop } = collect();
    const t = newTransport();
    t.start('2026');

    FakeEventSource.last!.open();
    expect(received).toEqual([{ type: SIGNAL_RESYNC, reason: 'connect' }]);

    // EventSource reconnects by itself; the second open must revalidate too,
    // because signals may have been missed while it was down.
    FakeEventSource.last!.open();
    expect(received[1]).toEqual({ type: SIGNAL_RESYNC, reason: 'reconnect' });

    stop();
  });

  it('republishes entity.changed signals onto the bus', () => {
    const { received, stop } = collect();
    const t = newTransport();
    t.start('2026');
    FakeEventSource.last!.open();
    received.length = 0; // drop the connect resync

    FakeEventSource.last!.emit(SIGNAL_ENTITY_CHANGED, {
      type: SIGNAL_ENTITY_CHANGED,
      entity: 'patrulje',
      id: 'p-1',
      year: '2026',
      event: 'started',
    });

    expect(received).toEqual([
      {
        type: SIGNAL_ENTITY_CHANGED,
        entity: 'patrulje',
        id: 'p-1',
        year: '2026',
        event: 'started',
      },
    ]);
    stop();
  });

  it('forwards a server-sent resync', () => {
    const { received, stop } = collect();
    const t = newTransport();
    t.start('2026');
    FakeEventSource.last!.open();
    received.length = 0;

    FakeEventSource.last!.emit(SIGNAL_RESYNC, { type: SIGNAL_RESYNC });

    expect(received[0].type).toBe(SIGNAL_RESYNC);
    stop();
  });

  it('survives a malformed frame', () => {
    const { received, stop } = collect();
    const t = newTransport();
    t.start('2026');
    FakeEventSource.last!.open();
    received.length = 0;

    // A bad frame must not take the stream down; the next signal or reconnect
    // resync brings the client back into line.
    expect(() => FakeEventSource.last!.emit(SIGNAL_ENTITY_CHANGED, 'not json')).not.toThrow();
    expect(received).toHaveLength(0);

    FakeEventSource.last!.emit(SIGNAL_ENTITY_CHANGED, {
      type: SIGNAL_ENTITY_CHANGED,
      entity: 'sos',
      id: 'c-1',
      year: '2026',
    });
    expect(received).toHaveLength(1);
    stop();
  });

  it('distinguishes retrying from having given up', () => {
    const t = newTransport();
    t.start('2026');
    FakeEventSource.last!.open();

    FakeEventSource.last!.fail(0); // CONNECTING — the browser is retrying
    expect(t.state.value).toBe('reconnecting');

    FakeEventSource.last!.fail(2); // CLOSED — it has stopped
    expect(t.state.value).toBe('offline');
  });

  it('starting twice does not open a second connection', () => {
    const t = newTransport();
    t.start('2026');
    const first = FakeEventSource.last;

    t.start('2026');

    expect(FakeEventSource.last).toBe(first);
  });

  it('stop closes the connection and goes offline', () => {
    const t = newTransport();
    t.start('2026');
    const es = FakeEventSource.last!;
    es.open();

    t.stop();

    expect(es.closed).toBe(true);
    expect(t.state.value).toBe('offline');
  });

  it('a restart after stop reports connect, not reconnect', () => {
    const { received, stop } = collect();
    const t = newTransport();

    t.start('2026');
    FakeEventSource.last!.open();
    t.stop();
    received.length = 0;

    t.start('2026');
    FakeEventSource.last!.open();

    // A deliberate restart (e.g. a year change) is a fresh subscription, not a
    // recovery from a dropped one.
    expect(received[0]).toEqual({ type: SIGNAL_RESYNC, reason: 'connect' });
    stop();
  });

  it('does not publish a signal with no entity', () => {
    const { received, stop } = collect();
    const t = newTransport();
    t.start('2026');
    FakeEventSource.last!.open();
    received.length = 0;

    // A signal naming no entity would invalidate nothing while looking like it
    // worked — the server rejects these, and the client should not invent them.
    FakeEventSource.last!.emit(SIGNAL_ENTITY_CHANGED, { type: SIGNAL_ENTITY_CHANGED, year: '2026' });

    expect(received).toHaveLength(0);
    stop();
  });
});

describe('entity-set announcement', () => {
  beforeEach(() => {
    resetKnownEntities();
  });

  it('routes the entities frame to the dependency checker, not the signal bus', () => {
    const c = collect();
    const t = newTransport();
    t.start('2026');
    FakeEventSource.last!.open();
    c.received.length = 0; // drop the connect resync

    FakeEventSource.last!.emit(SIGNAL_ENTITIES, { entities: ['klan', 'qr'], exhaustive: false });

    expect(knownEntities()).toEqual({ entities: ['klan', 'qr'], exhaustive: false });
    // It describes the stream rather than reporting a change, so nothing should
    // treat it as an invalidation.
    expect(c.received).toEqual([]);

    c.stop();
    t.stop();
  });

  it('ignores a malformed or shapeless frame rather than taking the stream down', () => {
    const t = newTransport();
    t.start('2026');
    FakeEventSource.last!.open();

    FakeEventSource.last!.emit(SIGNAL_ENTITIES, 'not json');
    expect(knownEntities()).toBeUndefined();

    FakeEventSource.last!.emit(SIGNAL_ENTITIES, { exhaustive: true });
    expect(knownEntities()).toBeUndefined();

    // Still working afterwards.
    FakeEventSource.last!.emit(SIGNAL_ENTITIES, { entities: [], exhaustive: true });
    expect(knownEntities()).toEqual({ entities: [], exhaustive: true });

    t.stop();
  });
});
