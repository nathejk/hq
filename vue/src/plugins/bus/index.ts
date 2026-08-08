import mitt from 'mitt';
import type { ToastMessageOptions } from 'primevue/toast';
import type { LiveSignal } from '@/plugins/live/signals';

/**
 * Application-wide event map for the mitt bus.
 *
 * Keep this small and typed: the bus is the seam between infrastructure
 * (the axios plugin, the live-update transport) and the components that react
 * to it, so an untyped payload here becomes an untyped payload everywhere.
 */
export type TBusEvent = {
  /** Error/notification toasts, emitted by the axios plugin. */
  toast: ToastMessageOptions;

  /** A live-update signal from the transport. See plugins/live. */
  live: LiveSignal;
};

const bus = mitt<TBusEvent>();

export default bus;
