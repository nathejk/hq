import { ref, readonly, type Ref } from 'vue';
import bus from '@/plugins/bus';
import { SIGNAL_RESYNC, type LiveSignal } from './signals';

/**
 * How current the client believes it is.
 *
 * Deliberately absent: any notion of "stale" or "still replaying". The API does
 * not serve until its projections are fully caught up (PRD 005), so a connected
 * client is talking to a caught-up API by construction. The only distinction an
 * operator ever needs is connected vs unavailable — never fresh vs stale.
 */
export type ConnectionState =
  /** Push transport connected; changes arrive as they happen. */
  | 'live'
  /** Lost the connection, trying to re-establish. */
  | 'reconnecting'
  /** No push transport; revalidating on an interval. Honest about not being live. */
  | 'polling'
  /** Cannot reach the server at all. */
  | 'offline';

/**
 * A source of live-update signals.
 *
 * The interface has to express two quite different capabilities, and that
 * asymmetry is the point: SSE knows *which* entity changed, while polling can
 * only say *something might have*. Encoding both here is what lets SSE replace
 * polling later without touching a single page.
 *
 * Implementations publish onto the mitt bus rather than to callers, so nothing
 * has to hold a reference to the transport in order to react to it.
 */
export interface LiveTransport {
  /** Human-readable name, for the connection indicator and logs. */
  readonly name: string;

  /** Current connection state; reactive and read-only to consumers. */
  readonly state: Readonly<Ref<ConnectionState>>;

  /**
   * Begin producing signals. Idempotent: calling it twice must not double up
   * timers or connections.
   *
   * @param year The selected event year. Passed at subscribe time because
   *   `EventSource` cannot set headers, so the `X-YearSlug` mechanism the rest
   *   of the SPA uses is unavailable to the stream.
   */
  start(year: string): void;

  /** Stop producing signals and release everything. Safe to call when stopped. */
  stop(): void;
}

/** Publish a signal to the application. The single way signals enter the app. */
export function emitSignal(signal: LiveSignal): void {
  bus.emit('live', signal);
}

export type PollingTransportOptions = {
  /** How often to ask consumers to revalidate. */
  intervalMs?: number;
};

/**
 * Interval-based transport: the Phase 1 implementation.
 *
 * It cannot know what changed, so every tick emits a single `resync` — the same
 * signal a reconnect or a server-side buffer overflow produces. Consumers
 * therefore need no polling-specific branch: one code path covers "revalidate
 * everything", whatever caused it.
 *
 * Reports `polling`, never `live`, because claiming real-time behaviour the
 * transport does not have is exactly the dishonesty the connection indicator
 * exists to prevent.
 */
export function createPollingTransport(options: PollingTransportOptions = {}): LiveTransport {
  const intervalMs = options.intervalMs ?? 5000;
  const state = ref<ConnectionState>('offline');

  let timer: ReturnType<typeof setInterval> | undefined;

  return {
    name: 'polling',
    state: readonly(state) as Readonly<Ref<ConnectionState>>,

    start(_year: string) {
      // The year is irrelevant to this implementation: it carries no
      // per-entity information, so a resync simply revalidates whatever the
      // cache currently holds — which is already year-correct, because the
      // cache is flushed when the year changes (task 025).
      void _year;

      if (timer !== undefined) return; // idempotent

      state.value = 'polling';
      timer = setInterval(() => {
        emitSignal({ type: SIGNAL_RESYNC, reason: 'poll' });
      }, intervalMs);
    },

    stop() {
      if (timer !== undefined) {
        clearInterval(timer);
        timer = undefined;
      }
      state.value = 'offline';
    },
  };
}
