<script setup lang="ts">
/**
 * The shell around the track map (PRD 011).
 *
 * Mounted **once**, in `App.vue`, and driven by `useTrackViewer` — the same shape as PrimeVue's toast
 * service. The position glyph appears in seven views; mounting a dialog in each would mean seven copies
 * of this state and a Leaflet map instantiated per list.
 *
 * A dialog rather than a route, deliberately: an operator working down a list of names should not lose
 * their place to look at one patrol's movement.
 *
 * All this component owns is the frame and the time window. The map lives in `TrackMapPanel`, which is
 * **remounted** whenever the target or the window changes — see its comment for why that is necessary
 * rather than merely tidy.
 */
import { computed, ref, watch } from 'vue'
import { useTrackViewer } from '@/composables/useTrackViewer'
import TrackMapPanel from '@/components/TrackMapPanel.vue'

const { target, open, close } = useTrackViewer()

/**
 * Window presets rather than two date pickers.
 *
 * 30 hours on one screen is a tangle, and the question an operator actually asks is "where were they
 * between 22:00 and 02:00?" — but they ask it as "recently" or "tonight", not in epoch milliseconds.
 * The window doubles as the fidelity control: less time means the point budget buys more detail.
 */
const windows = [
  { label: 'Seneste time', ms: 60 * 60 * 1000 },
  { label: 'Seneste 6 timer', ms: 6 * 60 * 60 * 1000 },
  { label: 'Seneste 12 timer', ms: 12 * 60 * 60 * 1000 },
  { label: 'Hele løbet', ms: 0 },
]

// Six hours rather than everything: it covers the stretch usually being asked about, and it keeps the
// first render legible.
const windowMs = ref(6 * 60 * 60 * 1000)

// A fresh panel per target and per window. The key also carries the moment the window was chosen, so
// picking the same preset twice re-reads rather than showing the same cached slice — "seneste time"
// means the last hour *now*.
const nonce = ref(0)
watch([windowMs, target], () => nonce.value++)

const panelKey = computed(() => {
  const t = target.value
  const id = !t ? 'none' : t.kind === 'patrulje' ? `p:${t.teamId}` : `u:${t.personId}`
  return `${id}:${windowMs.value}:${nonce.value}`
})

const heading = computed(() => {
  const t = target.value
  if (!t) return 'Spor'
  if (t.label) return `Spor · ${t.label}`
  return t.kind === 'patrulje' ? 'Patruljens spor' : 'Spor'
})

const subheading = computed(() =>
  target.value?.kind === 'patrulje'
    ? 'Alle deltagere der har været i patruljen, nuværende og tidligere, samt patruljens scanninger.'
    : '',
)
</script>

<template>
  <Dialog
    :visible="open"
    modal
    :header="heading"
    :style="{ width: '92vw', maxWidth: '1100px' }"
    @update:visible="(v: boolean) => !v && close()"
  >
    <div class="flex flex-col gap-3">
      <p v-if="subheading" class="text-xs text-gray-500 m-0">{{ subheading }}</p>

      <SelectButton
        v-model="windowMs"
        :options="windows"
        optionLabel="label"
        optionValue="ms"
        :allowEmpty="false"
        size="small"
      />

      <TrackMapPanel v-if="target" :key="panelKey" :target="target" :window-ms="windowMs" />
    </div>
  </Dialog>
</template>
