import { computed } from 'vue';
import { liveState, type ConnectionState } from '@/plugins/live';

/**
 * Whether the app is currently receiving updates, for display.
 *
 * Exposed as a composable so views never import the transport: the transport is
 * an implementation detail that changes in PRD 004 Phase 2 (polling → SSE), and
 * nothing in the UI should have to change with it.
 *
 * Note what is deliberately absent: any notion of *stale* data. The API does not
 * serve until its projections are fully caught up (PRD 005), so a connected
 * client is talking to a caught-up API by construction. The only thing an
 * operator needs to distinguish is connected from unavailable.
 */

/** Danish labels for each state. */
const LABELS: Record<ConnectionState, string> = {
  live: 'Live',
  reconnecting: 'Forbinder igen…',
  polling: 'Opdaterer periodisk',
  offline: 'Ingen forbindelse',
};

/** A short explanation, for a tooltip. */
const DESCRIPTIONS: Record<ConnectionState, string> = {
  live: 'Ændringer vises med det samme.',
  reconnecting: 'Forbindelsen blev afbrudt. Prøver at genoprette.',
  polling: 'Henter ændringer med få sekunders mellemrum.',
  offline: 'Kan ikke nå serveren. Viste data kan mangle nye ændringer. Prøver igen hvert par sekunder.',
};

export function useConnectionState() {
  const state = liveState();

  return {
    state,
    label: computed(() => LABELS[state.value]),
    description: computed(() => DESCRIPTIONS[state.value]),
    /** Healthy enough that the operator need not act. */
    isHealthy: computed(() => state.value === 'live' || state.value === 'polling'),
    /** Nothing is arriving at all — the case worth drawing attention to. */
    isDisconnected: computed(
      () => state.value === 'offline' || state.value === 'reconnecting',
    ),
  };
}
