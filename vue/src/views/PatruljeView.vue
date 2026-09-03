<script setup>
import { ref, computed, watch } from 'vue';
import { useToast } from 'primevue/usetoast';
import { FilterMatchMode } from '@primevue/core/api';
import { http } from '@/plugins/axios';
import { useLiveResource } from '@/composables/useLiveResource';
import { daymonthhhmm } from '@/composables/datefilters';
import { memberStatusBadge } from '@/composables/sos';
import PositionIndicator from '@/components/PositionIndicator.vue';

const props = defineProps({
    teamId: {type: String, required: false},
})

const toast = useToast();

// Keyed by team, so several patrols can be cached side by side and opening a
// previously visited one is instant.
//
// The team itself is depended on by *instance*, so another patrol's edit does not
// refetch this page. Its members and orders cannot be: a spejder event names the
// member id and an order event names the order id, never the team, so there is no
// instance to key on — and a newly added member has an id this client has never
// seen. Those are therefore type-level, which does mean any scout or payment change
// anywhere revalidates an open patrol page. That is one small request on a page an
// operator has deliberately opened, and the alternative is a member list that
// silently stops updating.
//
// 'sos' was added for the "Kontakt med nødtelefon" card (PRD 001): the cases arrive
// in this same payload, so the card needs no resource of its own — only this token,
// which is also type-level because a case opened for this patrol has an id this
// client has never seen.
const { data, pending, error, refresh } = useLiveResource(
  `patrulje:detail:${props.teamId}`,
  async () => {
    const response = await http.get('/patrulje/' + props.teamId);
    return {
      team: response.data.team || {},
      members: response.data.members || [],
      orders: response.data.orders || [],
      sosCases: response.data.sosCases || [],
      config: response.data.config || {},
    };
  },
  { dependsOn: [`patrulje:${props.teamId}`, 'spejder', 'order', 'payment', 'sos'] },
);

const patrulje = computed(() => data.value?.team ?? {});
const spejdere = computed(() => data.value?.members ?? []);
const orders = computed(() => data.value?.orders ?? []);
const sosCases = computed(() => data.value?.sosCases ?? []);
const config = computed(() => data.value?.config ?? {});

// --- member lifecycle status (PRD 006) ---
//
// Label, colour and glyph come from the one shared badge vocabulary, so this column and the
// nødtelefon's member rows cannot describe the same status differently — the short forms are
// what fit in a Tag, while the backend's long labels stay where an operator *chooses* a
// status, in the correction picker below.
//
// "Ikke startet" for an absent status is the honest reading: the member exists on the roster
// and the race has not claimed them yet. Before PRD 006 this column said that for
// *everybody*, regardless of status — which is what made the bug invisible.
const memberStatusLabel = (slug) => memberStatusBadge(slug).label;
const memberStatusSeverity = (slug) => memberStatusBadge(slug).severity;
const memberStatusIcon = (slug) => memberStatusBadge(slug).icon;

// Whether a member is active **for this patrol**, which is not the same as being active.
//
// The roster this page lists is the signup roster (`spejder.teamId`), but membership for
// every purpose that counts — strength, discontinuation, the SOS card — follows
// `currentTeamId`. So a member moved to another patrol is still listed here and is still
// `racing`, just not for this team. Treating them as active would contradict the
// "{n} i løbet" this same page shows, which counts only members whose current team is this
// one.
const isActiveHere = (member) =>
  member.status === 'racing' && (!member.currentTeamId || member.currentTeamId === props.teamId);

const movedAway = (member) =>
  !!member.currentTeamId && member.currentTeamId !== props.teamId;

// Dimmed rather than hidden: a member who has left the race — or left for another patrol —
// is still somebody an operator may be asked about. gray-500 rather than lighter, because
// these are names read off a screen at three in the morning.
const memberNameClass = (member) => (isActiveHere(member) ? '' : 'text-gray-500');

watch(error, (err) => {
  if (!err) return;
  console.log('patrulje load failed', err);
  toast.add({ severity: 'error', summary: 'Kunne ikke hente patruljen', life: 5000 });
});
const filters = ref({
    'global': {value: null, matchMode: FilterMatchMode.CONTAINS},
});
const selectedValue = ref(null);
const expandedRows = ref([]);
const onRowExpand = (event) => {
 console.log(expandedRows.value)
 //   toast.add({ severity: 'info', summary: 'Row Group Expanded', detail: 'Value: ' + event.data, life: 3000 });
};
const onRowCollapse = (event) => {
  //  toast.add({ severity: 'success', summary: 'Row Group Collapsed', detail: 'Value: ' + event.data, life: 3000 });
};

// --- the correction interface (PRD 006, task 084) ---
//
// When reality and the record disagree — a member was driven in and nobody wrote it down,
// a status was set on the wrong person — this is where it gets put right.
//
// Deliberately *not* on the SOS case card: a correction is not part of the call an
// operator is on, and being a different screen is a stronger separation than a
// differently-styled button would be. It is also why this is tucked into the expanded row
// rather than offered as a column action.
const correcting = ref({});
const correctionDraft = ref({});

// `finished` is absent from the backend's list, so it cannot be offered here either: only
// walking the route unaided earns it, and no correction may confer it.
const correctionOptions = computed(() => config.value.memberStatuses ?? []);

const startCorrection = (member) => {
  correctionDraft.value = { ...correctionDraft.value, [member.memberId]: member.status || '' };
  correcting.value = { ...correcting.value, [member.memberId]: true };
};

const cancelCorrection = (memberId) => {
  correcting.value = { ...correcting.value, [memberId]: false };
};

const savingCorrection = ref({});

// No sosId is sent: the server mints a case for the correction and closes it immediately,
// so the change is documented without the operator having to open one by hand. It surfaces
// in the "Kontakt med nødtelefon" card below.
const saveCorrection = async (member) => {
  const to = correctionDraft.value[member.memberId];
  if (to === undefined || to === member.status) {
    cancelCorrection(member.memberId);
    return;
  }
  savingCorrection.value = { ...savingCorrection.value, [member.memberId]: true };
  try {
    await http.put(`/member/${member.memberId}/status`, { status: to });
    correcting.value = { ...correcting.value, [member.memberId]: false };
    await refresh();
  } catch {
    // surfaced by the axios plugin; the row stays open so the operator can see what they
    // chose rather than having to find the member again
  } finally {
    savingCorrection.value = { ...savingCorrection.value, [member.memberId]: false };
  }
};

const linkToSignUp = () => {
    window.open("http://tilmelding.nathejk.dk/patrulje/" + patrulje.value.id, '_blank')
}

const formatAmount = (value, currency) => {
    if (value == null) return ''
    return (value / 100).toLocaleString('da-DK', { style: 'currency', currency: currency || 'DKK' })
}
// Shared rather than parsed here: the order and payment tables serve Go's
// time.Time text form, which Safari refuses to parse. See parseApiDate.
const formatDateTime = (value) => daymonthhhmm(value)
const statusLabel = (status) => (status === 'PAID' ? 'Betalt' : 'Åben')
const statusSeverity = (status) => (status === 'PAID' ? 'success' : 'warn')
</script>

<template>
    <div class="card" id="patruljer">
        <h1 class="font-nathejk text-2xl">{{ patrulje.number || '×' }} - {{ patrulje.name }}</h1>

        <Button label="Tilmelding" icon="pi pi-external-link" iconPos="right" @click="linkToSignUp" />

        <DataTable :value="spejdere" :loading="pending" sortMode="single" sortField="lok" :sortOrder="1" :stripedRows="true" :filters="filters"
            v-model:expandedRows="expandedRows" dataKey="id" @rowExpand="onRowExpand" @rowCollapse="onRowCollapse"
        >
            <Column expander />
            <Column field="name" header="Navn" sortable>
                <template #body="{data}">
                    <span :class="memberNameClass(data)">{{ data.name }}</span>
                    <PositionIndicator :person-id="data.memberId" class="ml-1" />
                    <!--
                      Made visible in the collapsed row, not only in the expanded one: a
                      member moved to another patrol is still listed here and still racing,
                      so without saying so the row looks active while the patrol's own
                      "{n} i løbet" does not count them — two true numbers that appear to
                      disagree.
                    -->
                    <span v-if="movedAway(data)" class="ml-2 text-xs italic text-gray-500">
                        flyttet til anden patrulje
                    </span>
                </template>
            </Column>
            <Column field="phone" header="Telefon" sortable></Column>
            <Column field="phoneParent" header="Kontaktperson"></Column>
            <Column field="status" header="Status">
                <template #body="{data}">
                    <Tag :value="memberStatusLabel(data.status)"
                         :icon="memberStatusIcon(data.status)"
                         :severity="memberStatusSeverity(data.status)" />
                </template>
            </Column>
            <template #expansion="{data}">
                <!--
                  The correction interface (PRD 006, task 084). This row used to render
                  `{{ data }}`, which is why there was room here.
                -->
                <div class="pl-2 py-1 text-sm">
                    <div class="mb-2">
                        <span class="text-gray-600">Status:</span>
                        <Tag class="ml-2" :value="memberStatusLabel(data.status)"
                             :icon="memberStatusIcon(data.status)"
                             :severity="memberStatusSeverity(data.status)" />
                        <span v-if="data.updatedAt" class="ml-2 text-gray-500">
                            ændret {{ formatDateTime(data.updatedAt) }}
                        </span>
                        <span v-if="data.currentTeamId && data.currentTeamId !== data.teamId"
                              class="ml-2 text-gray-500">
                            — flyttet til anden patrulje
                        </span>
                    </div>

                    <div v-if="!correcting[data.memberId]">
                        <Button label="Ret status manuelt" size="small" severity="secondary"
                                outlined icon="pi pi-wrench"
                                @click="startCorrection(data)" />
                        <div class="mt-1 text-gray-500">
                            Brug kun når virkeligheden ikke passer med det registrerede —
                            rettelsen dokumenteres automatisk som en sag.
                        </div>
                    </div>

                    <div v-else class="flex flex-wrap items-center gap-2">
                        <Select :model-value="correctionDraft[data.memberId]"
                                :options="correctionOptions" option-label="label"
                                option-value="slug" placeholder="Vælg status"
                                class="w-64"
                                @update:model-value="correctionDraft = { ...correctionDraft, [data.memberId]: $event }" />
                        <Button label="Gem rettelse" size="small"
                                :loading="savingCorrection[data.memberId]"
                                @click="saveCorrection(data)" />
                        <Button label="Annuller" size="small" severity="secondary" text
                                @click="cancelCorrection(data.memberId)" />
                    </div>
                </div>
            </template>
        </DataTable>

        <!--
          Kontakt med nødtelefon (PRD 001). Hidden entirely when the patrol has never
          called: most never do, and an empty card on every patrol page is noise that
          trains operators to ignore the place where a real incident would appear.
        -->
        <template v-if="sosCases.length">
            <h1 class="font-nathejk text-2xl mt-5">Kontakt med nødtelefon</h1>
            <DataTable :value="sosCases" :stripedRows="true" dataKey="id" selectionMode="single"
                @row-click="$router.push({ name: 'sos', params: { id: $event.data.id } })">
                <Column field="createdAt" header="Oprettet">
                    <template #body="{data}">{{ formatDateTime(data.createdAt) }}</template>
                </Column>
                <Column field="headline" header="Overskrift" />
                <Column field="status" header="Status">
                    <template #body="{data}">
                        <Tag :value="data.status === 'closed' ? 'Afsluttet' : 'Åben'"
                             :severity="data.status === 'closed' ? 'secondary' : 'info'" />
                    </template>
                </Column>
            </DataTable>
        </template>

        <h1 class="font-nathejk text-2xl mt-5">Betalinger</h1>
        <DataTable :value="orders" sortMode="single" sortField="createdAt" :sortOrder="-1" :stripedRows="true" >
            <template #empty>Ingen bestillinger</template>
            <Column field="createdAt" header="Tidspunkt" sortable>
                <template #body="{data}">{{ formatDateTime(data.createdAt) }}</template>
            </Column>
            <Column field="totalAmount" header="Beløb" sortable>
                <template #body="{data}">{{ formatAmount(data.totalAmount, data.currency) }}</template>
            </Column>
            <Column field="paidAmount" header="Betalt">
                <template #body="{data}">{{ formatAmount(data.paidAmount, data.currency) }}</template>
            </Column>
            <Column field="dueAmount" header="Mangler">
                <template #body="{data}">{{ formatAmount(data.dueAmount, data.currency) }}</template>
            </Column>
            <Column field="status" header="Status">
                <template #body="{data}">
                    <Tag :value="statusLabel(data.status)" :severity="statusSeverity(data.status)" />
                </template>
            </Column>
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
