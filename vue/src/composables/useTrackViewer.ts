// Opening the track map from anywhere a person's name appears (PRD 011).
//
// # Why a singleton rather than a dialog per view
//
// The position glyph sits in seven views and components, and clicking it should open a map. Mounting
// `<TrackMapDialog>` in each of them would mean seven copies of the same state, seven chances to wire
// it differently, and a Leaflet map instantiated per list. Instead one dialog is mounted in `App.vue`
// and this holds what it should show — the same shape PrimeVue's own toast service uses, and the same
// reasoning as `composables/kort.ts`: one owner for a thing that is read from many places.
//
// # Why the spejder rule lives here
//
// For a spejder the unit that matters is the **patrol**, not the person: scouts move between patrols,
// phones die, and one member's line answers almost nothing on its own. So clicking a scout's glyph
// opens the patrol map — everyone who has been on the team, plus its scans — while clicking a gøgler's
// or a crew member's opens their own track.
//
// That rule is expressed once, here, rather than at each of the seven call sites. A call site only
// says what it knows: this is a member of team X, or this is a person.

import { computed, ref } from 'vue'

export type TrackTarget =
  /** A patrol: every member who has ever been on it, current and former, plus its scans. */
  | { kind: 'patrulje'; teamId: string; label?: string }
  /** One person's own track — crew, gøgler, friend, bandit, or a senior. */
  | { kind: 'person'; personId: string; label?: string }

const target = ref<TrackTarget | undefined>()

export function useTrackViewer() {
  return {
    /** What to show, or undefined when the dialog is closed. */
    target: computed(() => target.value),
    open: computed(() => target.value !== undefined),

    /**
     * Open the map for a person.
     *
     * `teamId` is what decides which map appears: a scout on a patrol gets the patrol's picture. A
     * caller that has a team id passes it; one that does not gets the single-person track, which is
     * the right answer for crew and gøglere anyway.
     */
    show(personId: string, opts?: { teamId?: string; label?: string }) {
      if (opts?.teamId) {
        target.value = { kind: 'patrulje', teamId: opts.teamId, label: opts.label }
        return
      }
      target.value = { kind: 'person', personId, label: opts?.label }
    },

    close() {
      target.value = undefined
    },
  }
}
