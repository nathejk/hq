<script setup>
import { computed, watch } from 'vue';
import { http } from '@/plugins/axios';
import { useLiveResource } from '@/composables/useLiveResource';
import { daymonthhhmm } from '@/composables/datefilters';

const props = defineProps({
    teamId: {type: String, required: false},
})

// Read-only, and the one place in the SPA where a *scan* landing is the whole
// point: this is the running patrol's checkpoint trail. Hence the `qr` dependency
// (the subject is NATHEJK.*.qr.*.scanned; `scan` is only the table's name), which
// has to be type-level because a scan event names the qr, never the team. The
// instance dependency covers the start row, whose time lives on the patrulje.
//
// Cost, measured rather than assumed: this endpoint answers in ~3.4ms with a 5.5KB
// body, and the busiest minute in the existing scan data is 17 scans, so even a
// dozen expanded rows revalidating on every scan is negligible.
const { data, error } = useLiveResource(
  `patrulje:scans:${props.teamId}`,
  async () => {
    const response = await http.get('/patrulje/' + props.teamId + '/scans');
    return {
      team: response.data.team || {},
      scans: response.data.scans || [],
    };
  },
  { dependsOn: [`patrulje:${props.teamId}`, 'qr'] },
);

const patrulje = computed(() => data.value?.team ?? {});
const scans = computed(() => data.value?.scans ?? []);

// The scans in time order, with the start prepended as its own event. The endpoint
// already returns scans ordered by uts ascending, and the start is always the
// earliest, so a prepend keeps the trail chronological. The synthetic row carries
// `type: 'start'` so the template can label it and skip the position link a start
// has no coordinates for; scan rows carry the resolved `scanner` the server built.
const events = computed(() => {
  const rows = scans.value.map((s) => ({
    type: 'scan',
    uts: s.uts,
    scanner: s.scanner ?? { kind: 'unknown' },
    latitude: s.latitude,
    longitude: s.longitude,
  }));
  if (patrulje.value.startedUts) {
    rows.unshift({
      type: 'start',
      uts: patrulje.value.startedUts,
      label: `Startet · ${patrulje.value.memberCount} spejdere`,
    });
  }
  return rows;
});

// A bandit catches teams ("Fanget af"); everyone else scans them ("Scannet af"). The
// phone number is deliberately never shown — it is not something an operator can act
// on at a glance. A bandit reads as <number> <name> <klan>; crew as <section> <name>,
// with the post appended when they scanned while staffing one.
const eventVerb = (e) => (e.scanner?.kind === 'bandit' ? 'Fanget af' : 'Scannet af');
const scannerText = (s) => {
  if (!s) return 'ukendt';
  if (s.kind === 'bandit') {
    return [s.armNumber, s.name, s.klan].filter(Boolean).join(' · ') || 'ukendt bandit';
  }
  if (s.kind === 'crew') {
    const who = [s.sectionLabel, s.name].filter(Boolean).join(' · ');
    return s.checkpointName ? `${who} @ ${s.checkpointName}` : who || 'ukendt';
  }
  // Unknown scanner: the phone is all we have, but it beats "ukendt".
  return s.phone || 'ukendt';
};

// A scanner saves without a fix as "" or "0"; only a real pair gets a map link.
const hasPosition = (e) =>
  e.latitude && e.longitude && e.latitude !== '0' && e.longitude !== '0';
const mapsUrl = (e) => `https://www.google.com/maps?q=${e.latitude},${e.longitude}`;

watch(error, (err) => {
  if (err) console.log('patrulje scans load failed', err);
});
</script>

<template>
    <div class="card !bg-slate-300 pb-3" id="patruljer">
        <DataTable :value="events" class="!bg-transparent" size="small">
            <Column header="Tidspunkt">
                <template #body="{data}">
                    <span :class="data.type === 'start' ? 'font-semibold' : ''">{{ daymonthhhmm(data.uts * 1000) }}</span>
                </template>
            </Column>
            <Column header="Hændelse">
                <template #body="{data}">
                    <span v-if="data.type === 'start'" class="font-semibold">{{ data.label }}</span>
                    <span v-else><span class="text-slate-600">{{ eventVerb(data) }}:</span> {{ scannerText(data.scanner) }}</span>
                </template>
            </Column>
            <Column header="Position">
                <template #body="{data}">
                    <a v-if="hasPosition(data)" :href="mapsUrl(data)" target="_blank" rel="noopener">Kort</a>
                </template>
            </Column>
            <template #empty>
                <span class="text-slate-600">Ingen hændelser endnu</span>
            </template>
        </DataTable>
    </div>
</template>

<style>
#patruljer td {
    padding: 0.25rem 0.75rem;
}
@media (min-width: 1024px) {
  .about {
    min-height: 100vh;
    display: flex;
    align-items: center;
  }
}
</style>
