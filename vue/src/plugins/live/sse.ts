import { ref, readonly, type Ref } from 'vue';
import {
  emitSignal,
  type ConnectionState,
  type LiveTransport,
} from './transport';
import {
  SIGNAL_ENTITY_CHANGED,
  SIGNAL_RESYNC,
  type EntityChangedSignal,
} from './signals';

/**
 * Minimal shape of the EventSource API this transport uses.
 *
 * Declared rather than relying on the DOM global so the transport can be
 * constructed with a fake: unit tests run in a node environment, which has no
 * EventSource.
 */
export interface EventSourceLike {
  readyState: number;
  onopen: ((ev: unknown) => void) | null;
  onerror: ((ev: unknown) => void) | null;
  addEventListener(type: string, listener: (ev: { data: string }) => void): void;
  close(): void;
}

export type EventSourceFactory = (url: string) => EventSourceLike;

export type SseTransportOptions = {
  /** Endpoint path; the year is appended when the transport starts. */
  url?: string;

  /**
   * Entity tokens to subscribe to. Omitted means everything.
   *
   * Note the EventSource URL is fixed once opened, so changing this needs a
   * reconnect — which is why it is set here rather than derived from whatever
   * happens to be mounted.
   */
  entities?: string[];

  /** Injected for tests; defaults to the browser's EventSource. */
  create?: EventSourceFactory;
};

const CLOSED = 2;

/**
 * Live updates over Server-Sent Events.
 *
 * Chosen over a websocket because the client never sends anything — writes go
 * over REST — and because EventSource reconnects on its own. The legacy `dims`
 * channel hand-rolled backoff by re-entering a Vuex mutation and still could not
 * recover the messages it missed while down; here the browser handles retries and
 * every (re)connect emits a resync, so recovery is one line rather than a state
 * machine.
 *
 * Auth needs nothing: cookies are sent automatically same-origin, which covers
 * the basic auth in front of stage/prod today and the planned JWT cookie.
 */
export function createSseTransport(options: SseTransportOptions = {}): LiveTransport {
  const path = options.url ?? '/api/stream';
  const state = ref<ConnectionState>('offline');

  let source: EventSourceLike | undefined;
  let hasConnected = false;

  const create: EventSourceFactory =
    options.create ??
    ((url: string) => new EventSource(url, { withCredentials: true }) as unknown as EventSourceLike);

  const urlFor = (year: string) => {
    const params = new URLSearchParams();
    // Always sent explicitly, even when empty means "current year": relying on
    // the server's default would make a mismatch between the stream's year and
    // the year REST calls use invisible.
    params.set('year', year);
    if (options.entities?.length) {
      params.set('entities', options.entities.join(','));
    }
    return `${path}?${params.toString()}`;
  };

  return {
    name: 'sse',
    state: readonly(state) as Readonly<Ref<ConnectionState>>,

    start(year: string) {
      if (source) return; // idempotent

      state.value = 'reconnecting'; // not live until the connection opens
      const es = create(urlFor(year));
      source = es;

      es.onopen = () => {
        state.value = 'live';
        // Signals may have been missed while disconnected, so every connect —
        // including the first — revalidates. This is the same path polling and
        // hub overflow use, so there is one recovery behaviour, not three.
        emitSignal({
          type: SIGNAL_RESYNC,
          reason: hasConnected ? 'reconnect' : 'connect',
        });
        hasConnected = true;
      };

      es.onerror = () => {
        // EventSource retries by itself and reports CONNECTING while it does.
        // CLOSED means it has given up, which is the only case the operator
        // needs to act on.
        state.value = es.readyState === CLOSED ? 'offline' : 'reconnecting';
      };

      // Dispatch on the event name so a new kind of signal — a deploy
      // notification, say — is additive rather than a format change. Names we do
      // not know are simply never subscribed to, which is the same as ignoring
      // them.
      es.addEventListener(SIGNAL_ENTITY_CHANGED, (event) => {
        const signal = parse<EntityChangedSignal>(event.data);
        if (signal && signal.entity) {
          emitSignal({ ...signal, type: SIGNAL_ENTITY_CHANGED });
        }
      });

      es.addEventListener(SIGNAL_RESYNC, () => {
        emitSignal({ type: SIGNAL_RESYNC, reason: 'overflow' });
      });
    },

    stop() {
      source?.close();
      source = undefined;
      hasConnected = false;
      state.value = 'offline';
    },
  };
}

/**
 * Parse a payload, tolerating rubbish.
 *
 * A malformed frame must not take the stream down: the next signal, or the
 * resync that follows any reconnect, brings the client back into line.
 */
function parse<T>(data: string): T | undefined {
  try {
    return JSON.parse(data) as T;
  } catch {
    return undefined;
  }
}

/** Is a push transport usable in this environment? */
export function supportsEventSource(): boolean {
  return typeof EventSource !== 'undefined';
}
