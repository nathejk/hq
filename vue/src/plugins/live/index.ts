/**
 * Live updates: the SPA's single source of "something changed, refetch it".
 *
 * Structure:
 *   signals.ts    the wire vocabulary and dependency matching
 *   transport.ts  the transport interface + the polling implementation
 *   index.ts      the app-wide singleton and its lifecycle
 *
 * Pages never import from here. They compose `useLiveResource` (task 024),
 * which subscribes to the mitt bus. Keeping the transport private is what stops
 * the legacy mistake of every view knowing about the channel.
 */
import { createPollingTransport, type LiveTransport } from './transport';

export { createPollingTransport, emitSignal } from './transport';
export type { ConnectionState, LiveTransport } from './transport';
export * from './signals';

let transport: LiveTransport = createPollingTransport();
let startedYear: string | undefined;

/** The active transport. Swapped wholesale when SSE lands (PRD 004 Phase 2). */
export function liveTransport(): LiveTransport {
  return transport;
}

/**
 * Replace the transport implementation.
 *
 * Exists so Phase 2 can drop in SSE without touching any page, and so tests can
 * substitute a fake. Stops the current transport first, and restarts the new one
 * for the same year if one was running.
 */
export function setLiveTransport(next: LiveTransport): void {
  const year = startedYear;
  stopLive();
  transport = next;
  if (year !== undefined) startLive(year);
}

/** Start (or restart) signal delivery for a given event year. Idempotent. */
export function startLive(year: string): void {
  if (startedYear === year) return;
  if (startedYear !== undefined) transport.stop();
  startedYear = year;
  transport.start(year);
}

/** Stop signal delivery. Safe when already stopped. */
export function stopLive(): void {
  if (startedYear === undefined) return;
  transport.stop();
  startedYear = undefined;
}

/** The year the transport is currently subscribed to, if any. */
export function liveYear(): string | undefined {
  return startedYear;
}

/** Reactive connection state, for the indicator (task 026). */
export function liveState() {
  return transport.state;
}
