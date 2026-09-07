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

// --- the pre-race reshuffle (PRD 012) ---
//
// A contact person who paid for two teams wants to move somebody between them. The
// paid seat and any t-shirt move with the member, as an internal transfer between the
// two teams' orders — no card is charged and nothing is refunded.
//
// Deliberately a different action from the nødtelefon's "Skift patrulje": that one is
// for a member who *started* here and continues elsewhere, and it keeps them on this
// roster. This one changes which team they belong to, so they leave this list entirely.
// Hence it is only offered before the patrol has started.
const canReassign = computed(() => patrulje.value.status !== 'STARTED');

// The dialog's whole state in one ref, so closing it cannot leave a stale destination
// or a spinner behind.
const moveDlg = ref(null);

const openMove = async (member) => {
  moveDlg.value = {
    memberId: member.memberId,
    name: member.name,
    query: '',
    target: null,
    loading: true,
    submitting: false,
    candidates: [],
    lines: [],
    amount: 0,
    pendingTransfer: false,
  };
  try {
    const response = await http.get(`/member/${member.memberId}/transfer-candidates`);
    if (!moveDlg.value || moveDlg.value.memberId !== member.memberId) return;
    moveDlg.value = {
      ...moveDlg.value,
      loading: false,
      candidates: response.data.candidates ?? [],
      lines: response.data.lines ?? [],
      amount: response.data.amount ?? 0,
      pendingTransfer: response.data.pendingTransfer ?? false,
    };
  } catch {
    // surfaced by the axios plugin
    moveDlg.value = null;
  }
};

// Eligibility is the server's answer, not recomputed here: the same rule decides what
// this list offers and what the move accepts, so the two cannot disagree about whether
// a team is full.
//
// Ineligible teams are shown struck through rather than filtered out — an operator
// looking for a specific patrol needs to see that it is full or has started, not to
// wonder whether they misremembered the name.
const moveTargets = computed(() => {
  const d = moveDlg.value;
  if (!d) return [];
  const q = d.query.trim().toLowerCase();
  const all = d.candidates;
  if (q.length < 2) return all.slice(0, 10);
  return all
    .filter((c) => [c.teamNumber, c.name, c.group].some((f) => (f ?? '').toLowerCase().includes(q)))
    .slice(0, 10);
});

const submitMove = async () => {
  const d = moveDlg.value;
  if (!d || !d.target) return;
  moveDlg.value = { ...d, submitting: true };
  try {
    await http.put(`/member/${d.memberId}/reassign`, { teamId: d.target.teamId });
    const moved = d.name;
    const to = d.target;
    moveDlg.value = null;
    toast.add({
      severity: 'success',
      summary: `${moved} er flyttet`,
      detail: to.teamNumber ? `til patrulje ${to.teamNumber}` : `til ${to.name}`,
      life: 5000,
    });
    await refresh();
  } catch {
    // surfaced by the axios plugin; the dialog stays open with the chosen destination
    moveDlg.value = moveDlg.value ? { ...moveDlg.value, submitting: false } : null;
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

// A transfer's two orders are recognisable by their line ids (PRD 012): every line the
// transfer creates is `transfer:{transferId}:{n}`. Labelling them matters because they
// otherwise read as anonymous sales — and the credit side reads as a *negative* one,
// which looks like a mistake rather than money moving to another team.
//
// Detected from the lines rather than from a column on the order: the ids are already
// deterministic and carry the transfer, so nothing extra had to be stored for this.
const isTransfer = (order) => (order.lines ?? []).some((l) => (l.lineId ?? '').startsWith('transfer:'))
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
                    <!-- teamId given, so this opens the patrol's map rather than one member's line. -->
                    <PositionIndicator
                        :person-id="data.memberId"
                        :team-id="props.teamId"
                        :label="patrulje.name || data.name"
                        class="ml-1"
                    />
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
                        <!--
                          The pre-race reshuffle (PRD 012). Hidden rather than disabled once
                          the patrol has started: at that point the action does not exist
                          here at all, and the move belongs to the nødtelefon's case-based
                          one, so a greyed-out button would only invite a wrong question.
                        -->
                        <Button v-if="canReassign" label="Flyt til anden patrulje" size="small"
                                severity="secondary" outlined icon="pi pi-arrow-right-arrow-left"
                                class="ml-2"
                                @click="openMove(data)" />
                        <div class="mt-1 text-gray-500">
                            Brug kun når virkeligheden ikke passer med det registrerede —
                            rettelsen dokumenteres automatisk som en sag.
                        </div>
                        <div v-if="canReassign" class="text-gray-500">
                            Ved flytning følger den betalte plads og eventuelt t-shirt med over
                            i den nye patrulje.
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
                <template #body="{data}">
                    {{ formatAmount(data.totalAmount, data.currency) }}
                    <Tag v-if="isTransfer(data)" class="ml-2" severity="info"
                         value="Intern overførsel" />
                </template>
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

        <!--
          Flyt til anden patrulje (PRD 012). The team picker follows the same idiom as the
          nødtelefon's: a text filter over a short list with a "Vælg" button per row, not an
          AutoComplete — an operator reads a number off a screen and needs to see the
          candidates, including the ones they may not pick.
        -->
        <Dialog v-if="moveDlg" :visible="true" modal header="Flyt til anden patrulje"
                :style="{ width: '32rem' }" @update:visible="moveDlg = null">
            <p class="mb-3 text-sm">
                <strong>{{ moveDlg.name }}</strong> flyttes til en anden patrulje og hører
                derefter til den. Den betalte plads og eventuelle køb følger med.
            </p>

            <div v-if="moveDlg.loading" class="text-sm text-gray-500">Henter patruljer …</div>

            <template v-else>
                <!-- What moves, priced as it was actually paid for. -->
                <div class="mb-3 rounded bg-gray-50 p-2 text-sm">
                    <!--
                      A previous transfer that has not settled yet holds this member's seat,
                      so the amounts below cannot be trusted and a move must wait. Said
                      before them, because the important fact is that the money exists — the
                      alternative wording ("der er ikke betalt") would be a lie (task 166).
                    -->
                    <template v-if="moveDlg.pendingTransfer">
                        <span class="font-semibold text-orange-700">
                            En tidligere flytning af {{ moveDlg.name }} er ikke afregnet endnu.
                        </span>
                        <div class="mt-1 text-gray-600">
                            Den betalte plads er på vej over, men er ikke bogført færdig — prøv
                            igen om et øjeblik. Flytter vi nu, bliver betalingen efterladt hos
                            den forrige patrulje.
                        </div>
                    </template>
                    <template v-else-if="moveDlg.lines.length">
                        <div v-for="l in moveDlg.lines" :key="l.productSku + (l.size || '')"
                             class="flex justify-between">
                            <span>
                                {{ l.productName }}
                                <span v-if="l.size" class="text-gray-500">({{ l.size }})</span>
                                <span v-if="l.quantity !== 1" class="text-gray-500">× {{ l.quantity }}</span>
                            </span>
                            <span>{{ formatAmount(l.lineTotal) }}</span>
                        </div>
                        <div class="mt-1 flex justify-between border-t border-gray-200 pt-1 font-semibold">
                            <span>Overføres som intern overførsel</span>
                            <span>{{ formatAmount(moveDlg.amount) }}</span>
                        </div>
                    </template>
                    <!--
                      Nothing to move is an ordinary outcome, not an error: the team has not
                      paid for this member yet. Said out loud so the operator is not left
                      wondering why no money moved.
                    -->
                    <span v-else class="text-gray-600">
                        Der er ikke betalt for denne deltager, så der overføres ingen betaling.
                    </span>
                </div>

                <template v-if="!moveDlg.pendingTransfer">
                <div v-if="moveDlg.target" class="mb-2 flex items-center gap-2">
                    <strong>
                        <span v-if="moveDlg.target.teamNumber">{{ moveDlg.target.teamNumber }} · </span>
                        {{ moveDlg.target.name }}
                    </strong>
                    <Button label="Skift" text size="small" @click="moveDlg = { ...moveDlg, target: null }" />
                </div>
                <div v-else>
                    <InputText :model-value="moveDlg.query" class="w-full"
                               placeholder="Søg patrulje (nummer, navn, gruppe)"
                               @update:model-value="moveDlg = { ...moveDlg, query: $event ?? '' }" />
                    <div v-for="c in moveTargets" :key="c.teamId"
                         class="flex items-center justify-between gap-2 border-b border-gray-100 py-1 last:border-0">
                        <span class="text-sm" :class="c.eligible ? '' : 'text-gray-400 line-through'">
                            <span v-if="c.teamNumber">{{ c.teamNumber }} · </span>
                            {{ c.name }}
                            <span class="text-gray-500">{{ c.group }}</span>
                            <span class="text-gray-500">{{ c.memberCount }}/7</span>
                        </span>
                        <Button v-if="c.eligible" label="Vælg" size="small"
                                @click="moveDlg = { ...moveDlg, target: c }" />
                        <span v-else class="text-xs italic text-gray-400">{{ c.reason }}</span>
                    </div>
                    <small v-if="!moveTargets.length" class="text-gray-500">Ingen patruljer fundet</small>
                </div>
                </template>
            </template>

            <template #footer>
                <div class="flex justify-end gap-2">
                    <Button label="Annuller" severity="secondary" text @click="moveDlg = null" />
                    <Button label="Flyt" :disabled="!moveDlg.target || moveDlg.pendingTransfer"
                            :loading="moveDlg.submitting"
                            @click="submitMove" />
                </div>
            </template>
        </Dialog>
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
