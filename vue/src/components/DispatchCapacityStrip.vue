<script setup lang="ts">
// The capacity strip (PRD 009 §7): who can actually drive, right now.
//
// It sits above both panes because it is what makes an estimate legible instead of magic. A
// queued task's *anslået* time is derived from when a unit next comes on duty, and a number like
// that has to be explainable at 3am — "22.35, because Bil 2 comes on at 22.00" is a plan; "22.35"
// on its own is a guess the desk cannot defend.
//
// It is also where a unit that is not capacity says so. A dispatchable subsection with no car or
// nobody in it would otherwise sit in the tour picker looking available.

import { computed } from 'vue'
import {
  type Duty,
  type Unit,
  formatUtsTime,
  nextDutyStart,
  unitReadiness,
  unitsOnDuty,
} from '@/composables/dispatch'

const props = defineProps<{
  units: Unit[]
  duty: Duty[]
  nowMs: number
}>()

const onDuty = computed(() => unitsOnDuty(props.duty, props.nowMs))
const next = computed(() => nextDutyStart(props.duty, props.nowMs))

/** This unit's window covering now, so the strip can show when it ends. */
const currentWindow = (unit: Unit) => {
  const uts = props.nowMs / 1000
  return props.duty.find(
    (w) => w.sectionSlug === unit.sectionSlug && w.startUts <= uts && uts < w.endUts,
  )
}

/** And, when it is off, when it comes on. */
const nextWindow = (unit: Unit) => {
  const uts = props.nowMs / 1000
  return props.duty
    .filter((w) => w.sectionSlug === unit.sectionSlug && w.startUts > uts)
    .sort((a, b) => a.startUts - b.startUts)[0]
}
</script>

<template>
  <section class="border rounded p-2 bg-white">
    <div class="flex flex-wrap items-center gap-2">
      <h2 class="text-xs uppercase tracking-wide text-gray-500 pr-2">Enheder</h2>

      <!--
        No unit configured, and no unit on duty, are different facts and read differently. The
        first is something to fix on the Organisation page; the second is simply the small hours.
      -->
      <span v-if="units.length === 0" class="text-sm text-gray-600">
        Ingen kørsels-enheder. Marker en underafdeling som kørsels-enhed på Organisation.
      </span>
      <span v-else-if="onDuty.size === 0" class="text-sm text-gray-700">
        <template v-if="next">Næste enhed på vagt {{ formatUtsTime(next) }}</template>
        <template v-else>Ingen enheder på vagt, og ingen vagter aftalt</template>
      </span>
    </div>

    <div v-if="units.length" class="flex flex-wrap gap-2 pt-2">
      <div
        v-for="unit in units"
        :key="unit.sectionSlug"
        class="border rounded px-2 py-1 text-sm min-w-48"
        :class="onDuty.has(unit.sectionSlug) ? 'border-green-400 bg-green-50' : 'bg-gray-50'"
      >
        <div class="flex items-center gap-2">
          <i class="pi pi-truck" :class="onDuty.has(unit.sectionSlug) ? 'text-green-700' : 'text-gray-400'" />
          <span class="font-medium flex-1 truncate">{{ unit.label }}</span>
          <Tag
            :value="onDuty.has(unit.sectionSlug) ? 'på vagt' : 'ikke på vagt'"
            :severity="onDuty.has(unit.sectionSlug) ? 'success' : 'secondary'"
          />
        </div>

        <div class="text-xs text-gray-600">
          <template v-if="unit.vehicles.length">
            {{ unit.vehicles[0].licensePlate }}
            <span v-if="unit.vehicles[0].seatCount"> · {{ unit.vehicles[0].seatCount }} pladser</span>
          </template>
          <span v-else class="text-amber-700">intet køretøj</span>
        </div>

        <!-- The people, driver first if the vehicle names one. -->
        <div class="text-xs text-gray-600 truncate">
          <template v-if="unit.people.length">
            {{ unit.people.map((p) => p.name).join(', ') }}
          </template>
          <span v-else class="text-amber-700">ingen mandskab</span>
        </div>

        <div class="text-xs text-gray-600 tabular-nums">
          <template v-if="currentWindow(unit)">
            til {{ formatUtsTime(currentWindow(unit)!.endUts) }}
          </template>
          <template v-else-if="nextWindow(unit)">
            fra {{ formatUtsTime(nextWindow(unit)!.startUts) }}
          </template>
          <span v-else class="italic">ingen vagt aftalt</span>
        </div>

        <!--
          Not ready, and why. A unit missing a car or a crew is not capacity, and saying which is
          missing turns it into something somebody can fix.
        -->
        <p v-if="!unitReadiness(unit).ready" class="text-xs text-amber-700">
          Ikke klar: {{ unitReadiness(unit).missing.join(' og ') }}
        </p>
        <!-- A configuration mistake, flagged rather than forbidden: the desk can still work. -->
        <p v-if="unitReadiness(unit).tooManyVehicles" class="text-xs text-amber-700">
          {{ unit.vehicles.length }} køretøjer i samme enhed — ret det på Organisation
        </p>
      </div>
    </div>
  </section>
</template>
