<script setup lang="ts">
// One klan, in full: who they are, who is on the team, what they have paid — and the two
// administrative acts that belong to a klan rather than to a LOK arrangement, namely
// withdrawing it and overriding its status.
//
// # Why a status override exists
//
// The signup status is normally a *consequence*: a signup puts a klan on hold, a paid order
// moves it to PAID. That is correct, and it is why no such control existed. But the money
// does not always arrive the way the system expects — a bank transfer, cash at a meeting, one
// group paying for two — and no MobilePay callback will ever say so. The klan has genuinely
// paid and is treated all weekend as if it had not.
//
// The override changes the status and nothing else. It deliberately does not invent a payment:
// the orders and payments below stay exactly as they were, so an out-of-band status reads as
// what it is — an HQ decision — rather than as forged evidence. That is also why the money is
// shown here at all, right next to the control: it is the evidence for or against the act.

import { computed, ref } from 'vue'
import { useToast } from 'primevue/usetoast'
import { http } from '@/plugins/axios'
import { useLiveResource } from '@/composables/useLiveResource'
import { parseApiDate } from '@/composables/datefilters'

const props = defineProps<{
  teamId: string
  /** Fallback name, shown until the payload arrives. */
  name?: string
}>()

const emit = defineEmits<{
  close: []
  /**
   * The klan was withdrawn or its status changed.
   *
   * The host is told rather than left to its own live signal because it may be an editor
   * holding unsaved work — the bandit page defers incoming payloads while its LOK arrangement
   * is dirty, and a klan deleted from here has to leave that arrangement regardless.
   */
  changed: [{ deleted: boolean }]
}>()

type StatusOption = { slug: string; label: string }

interface KlanDetail {
  team: {
    id: string
    year: string
    status: string
    name: string
    group: string
    korps: string
    memberCount: number
    lok: string
    paidAmount: number
  }
  members: {
    memberId: string
    name: string
    address: string
    postalCode: string
    city: string
    email: string
    phone: string
    birthday: string | null
    tshirtSize: string
    diet: string
    armNumber: string
  }[]
  signup: {
    name: string
    email: string | null
    emailPending: string
    phone: string | null
    phonePending: string
    createdAt: string
  } | null
  orders: {
    orderId: string
    status: string
    totalAmount: number
    paidAmount: number
    dueAmount: number
    createdAt: string
    lines: { lineId: string; productName: string; quantity: number; unitPrice: number; lineTotal: number }[]
  }[]
  payments: {
    reference: string
    amount: number
    method: string
    status: string
    createdAt: string
  }[]
  statusOptions: StatusOption[]
}

const toast = useToast()

// Live, keyed per klan so reopening one renders from cache.
//
// `klan:{id}` is the instance dependency — this klan's own events, including the status
// change this dialog publishes, so the badge corrects itself without a manual refetch. The
// rest are type-level because they are joined-in collections whose rows have ids this client
// has never seen: a member added elsewhere, an order line, a payment arriving from the
// provider. Tokens are the event subjects' entities (senior, order, payment), not the names
// used here.
const { data, pending, error, refresh } = useLiveResource(
  `klan:${props.teamId}`,
  async () => {
    const response = await http.get(`/klan/${props.teamId}`)
    return response.data as KlanDetail
  },
  { dependsOn: [`klan:${props.teamId}`, 'senior', 'order', 'payment'] },
)

const detail = computed<KlanDetail | null>(() => data.value ?? null)
const team = computed(() => detail.value?.team ?? null)

// The collections, normalised once.
//
// The template reads these and never the payload's own arrays, because a missing
// collection must not be able to break this dialog: a nil slice from the API
// marshals to `null`, `null.length` throws during render, and a render throw takes
// the dialog's own close button with it — leaving the operator stuck in a modal
// with no way out but a reload. That is exactly what a klan with no seniors did
// (no members, so no order lines, so no order at all).
//
// The server no longer sends null for these, but the guarantee is worth having on
// both sides: the cost is four lines, and the failure it prevents is the worst
// kind this screen can produce.
const members = computed(() => detail.value?.members ?? [])
const orders = computed(() => detail.value?.orders ?? [])
const payments = computed(() => detail.value?.payments ?? [])
const statusOptions = computed(() => detail.value?.statusOptions ?? [])

const busy = ref(false)

/** The status the operator has selected but not yet applied. */
const statusDraft = ref<string | null>(null)

/**
 * The status shown in the picker: the operator's choice, else the klan's own.
 *
 * Falling back to the live value rather than snapshotting it on open is what lets the picker
 * follow a change made elsewhere while the dialog is open, instead of quietly offering to
 * re-apply a status that is no longer current.
 */
const selectedStatus = computed({
  get: () => statusDraft.value ?? team.value?.status ?? null,
  set: (value: string | null) => (statusDraft.value = value),
})

const statusDirty = computed(() => !!selectedStatus.value && selectedStatus.value !== team.value?.status)

const statusLabel = (slug?: string) =>
  statusOptions.value.find((o) => o.slug === slug)?.label || slug || '—'

/**
 * Colour by how far through the lifecycle a status is, not by whether it is "good".
 *
 * Paid and started are settled, an unpaid klan is the one needing attention, and a withdrawn
 * one is neither — a colour scheme that flags OUT as an error would have the bandit chiefs
 * chasing klans that correctly left.
 */
const statusSeverity = (slug?: string) => {
  switch (slug) {
    case 'PAID':
    case 'STARTED':
      return 'success'
    case 'SEMIPAID':
      return 'warn'
    case 'PAY':
    case 'HOLD':
      return 'danger'
    case 'OUT':
      return 'secondary'
    default:
      return 'info'
  }
}

// Amounts are stored in øre; every screen that shows money divides by 100.
const kr = (oere: number) =>
  ((oere ?? 0) / 100).toLocaleString('da-DK', { style: 'currency', currency: 'DKK', maximumFractionDigits: 2 })

const date = (value?: string | null) => {
  const parsed = parseApiDate(value ?? undefined)
  return parsed ? parsed.toLocaleDateString('da-DK', { day: 'numeric', month: 'long', year: 'numeric' }) : ''
}

const paymentMethod = (method: string) => (method === 'mobilepay' ? 'MobilePay' : method || '—')
const paymentStatus = (status: string) => {
  switch (status) {
    case 'received':
      return 'Modtaget'
    case 'reserved':
      return 'Reserveret'
    case 'requested':
      return 'Anmodet'
    case 'cancelled':
      return 'Annulleret'
    default:
      return status
  }
}

/**
 * What the payments add up to versus what the orders ask for.
 *
 * Computed here rather than taken from a single order because a klan can have several, and the
 * question the operator is answering — "have these people actually paid?" — is about the total.
 */
const orderTotal = computed(() => orders.value.reduce((sum, o) => sum + (o.totalAmount ?? 0), 0))
const outstanding = computed(() => orderTotal.value - (team.value?.paidAmount ?? 0))

const errorDetail = (err: any) => err?.response?.data?.error ?? String(err)

async function saveStatus() {
  const next = selectedStatus.value
  if (!next || !statusDirty.value) return
  busy.value = true
  try {
    await http.patch(`/klan/${props.teamId}`, { status: next })
    toast.add({
      severity: 'success',
      summary: 'Status ændret',
      detail: `${team.value?.name || 'Klan'} → ${statusLabel(next)}`,
      life: 3000,
    })
    // Cleared so the picker follows the payload again rather than holding the operator's
    // choice as a permanent local override.
    statusDraft.value = null
    await refresh()
    emit('changed', { deleted: false })
  } catch (err: any) {
    toast.add({ severity: 'error', summary: 'Status blev ikke ændret', detail: errorDetail(err), life: 5000 })
  } finally {
    busy.value = false
  }
}

async function remove() {
  const label = team.value?.name || props.name || 'denne klan'
  // A klan carries members and money, so this asks in words rather than with an icon: the
  // count is in the question because deleting a klan of four is not the same decision as
  // deleting an empty one.
  const count = team.value?.memberCount ?? 0
  if (!window.confirm(`Slet klanen "${label}" med ${count} ${count === 1 ? 'senior' : 'seniorer'}?`)) return
  busy.value = true
  try {
    await http.delete(`/klan/${props.teamId}`)
    toast.add({ severity: 'success', summary: 'Klan slettet', detail: label, life: 3000 })
    emit('changed', { deleted: true })
    emit('close')
  } catch (err: any) {
    toast.add({ severity: 'error', summary: 'Klanen blev ikke slettet', detail: errorDetail(err), life: 5000 })
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <Dialog
    :visible="true"
    modal
    maximizable
    :style="{ width: '54rem' }"
    :breakpoints="{ '1199px': '85vw', '575px': '95vw' }"
    @update:visible="emit('close')"
  >
    <template #header>
      <div class="inline-flex items-center gap-2 text-2xl">
        <i class="fas fa-fw fa-users"></i>
        <h1 class="font-nathejk">{{ team?.name || name || 'Klan' }}</h1>
        <Tag v-if="team" :value="statusLabel(team.status)" :severity="statusSeverity(team.status)" class="text-base" />
      </div>
    </template>

    <!-- Only when nothing is cached: reopening a klan must not flash. -->
    <div v-if="pending && !detail" class="text-sm text-gray-500">Henter…</div>
    <div v-else-if="error && !detail" class="text-sm text-red-600">Kunne ikke hente klanen.</div>

    <template v-if="detail && team">
      <Fieldset legend="Oplysninger" class="mb-4">
        <dl class="grid grid-cols-[10rem_1fr] items-baseline gap-x-3 gap-y-1 text-sm">
          <dt class="text-gray-500">Gruppe / Division</dt>
          <dd>{{ team.group || '—' }}</dd>

          <dt class="text-gray-500">Korps</dt>
          <dd>{{ team.korps || '—' }}</dd>

          <dt class="text-gray-500">LOK</dt>
          <dd>{{ team.lok || 'ikke tildelt' }}</dd>

          <dt class="text-gray-500">Seniorer</dt>
          <dd>{{ team.memberCount }}</dd>

          <dt class="text-gray-500">År</dt>
          <dd>{{ team.year || '—' }}</dd>

          <template v-if="detail.signup">
            <dt class="text-gray-500">Tilmeldt af</dt>
            <dd>{{ detail.signup.name || '—' }}</dd>

            <dt class="text-gray-500">Kontakt</dt>
            <dd class="flex flex-wrap gap-x-4">
              <a v-if="detail.signup.phone || detail.signup.phonePending" :href="`tel:${detail.signup.phone || detail.signup.phonePending}`" class="underline">
                {{ detail.signup.phone || detail.signup.phonePending }}
              </a>
              <a v-if="detail.signup.email || detail.signup.emailPending" :href="`mailto:${detail.signup.email || detail.signup.emailPending}`" class="underline">
                {{ detail.signup.email || detail.signup.emailPending }}
              </a>
            </dd>

            <dt class="text-gray-500">Tilmeldt</dt>
            <dd>{{ date(detail.signup.createdAt) || '—' }}</dd>
          </template>
        </dl>

        <!-- The signup page, as on the patrol page: the same klan, seen as the klan sees it. -->
        <div class="pt-3">
          <a :href="`https://tilmelding.nathejk.dk/klan/${teamId}`" target="_blank" rel="noopener">
            <Button label="Tilmelding" icon="pi pi-external-link" iconPos="right" size="small" text />
          </a>
        </div>
      </Fieldset>

      <Fieldset :legend="`Seniorer (${members.length})`" class="mb-4">
        <DataTable :value="members" size="small" class="text-sm">
          <Column field="name" header="Navn" />
          <Column header="Kontakt">
            <template #body="{ data: m }">
              <div class="flex flex-col">
                <a v-if="m.phone" :href="`tel:${m.phone}`" class="underline">{{ m.phone }}</a>
                <a v-if="m.email" :href="`mailto:${m.email}`" class="underline text-xs">{{ m.email }}</a>
              </div>
            </template>
          </Column>
          <Column header="Adresse">
            <template #body="{ data: m }">
              <span v-if="m.address">{{ m.address }}<span v-if="m.postalCode || m.city">, {{ m.postalCode }} {{ m.city }}</span></span>
              <span v-else class="text-gray-400">—</span>
            </template>
          </Column>
          <Column header="Fødselsdag">
            <template #body="{ data: m }">{{ date(m.birthday) || '—' }}</template>
          </Column>
          <Column header="T-shirt">
            <template #body="{ data: m }">{{ (m.tshirtSize || '—').toUpperCase() }}</template>
          </Column>
          <!-- Kost is why a senior may not be able to eat what the kitchen cooked, so it is a
               column rather than something to open a row for. -->
          <Column header="Kost">
            <template #body="{ data: m }">{{ m.diet || '—' }}</template>
          </Column>
          <Column header="Banditnr.">
            <template #body="{ data: m }">{{ m.armNumber || '—' }}</template>
          </Column>
          <template #empty>
            <span class="text-gray-500">Ingen seniorer tilmeldt.</span>
          </template>
        </DataTable>
      </Fieldset>

      <!--
        The money, above the override it justifies. Orders are what was asked for, payments what
        arrived; showing only the total would hide exactly the discrepancy that makes an operator
        reach for the override in the first place.
      -->
      <Fieldset legend="Betaling" class="mb-4">
        <dl class="grid grid-cols-[10rem_1fr] items-baseline gap-x-3 gap-y-1 text-sm pb-3">
          <dt class="text-gray-500">Opkrævet</dt>
          <dd>{{ kr(orderTotal) }}</dd>

          <dt class="text-gray-500">Registreret betalt</dt>
          <dd>{{ kr(team.paidAmount) }}</dd>

          <dt class="text-gray-500">Mangler</dt>
          <dd :class="outstanding > 0 ? 'text-red-600 font-medium' : 'text-gray-500'">
            {{ outstanding > 0 ? kr(outstanding) : 'intet' }}
          </dd>
        </dl>

        <DataTable v-if="payments.length" :value="payments" size="small" class="text-sm">
          <Column header="Beløb">
            <template #body="{ data: p }">{{ kr(p.amount) }}</template>
          </Column>
          <Column header="Metode">
            <template #body="{ data: p }">{{ paymentMethod(p.method) }}</template>
          </Column>
          <Column header="Status">
            <template #body="{ data: p }">{{ paymentStatus(p.status) }}</template>
          </Column>
          <Column header="Dato">
            <template #body="{ data: p }">{{ date(p.createdAt) || '—' }}</template>
          </Column>
        </DataTable>
        <p v-else class="text-sm text-gray-500">
          Ingen betalinger registreret gennem systemet.
        </p>

        <div v-if="orders.length" class="pt-3 text-sm">
          <div v-for="o in orders" :key="o.orderId" class="pt-2">
            <div class="text-gray-500">
              Ordre · {{ kr(o.totalAmount) }} · mangler {{ kr(o.dueAmount) }} · {{ date(o.createdAt) }}
            </div>
            <ul class="pl-4 list-disc">
              <!-- `?? []` for the same reason as the collections above: an order whose
                   lines have not been projected yet must not break the render. -->
              <li v-for="l in o.lines ?? []" :key="l.lineId">
                {{ l.quantity }} × {{ l.productName }} — {{ kr(l.lineTotal) }}
              </li>
            </ul>
          </div>
        </div>
      </Fieldset>

      <Fieldset legend="Status" class="mb-2">
        <p class="text-sm text-gray-500 pb-3">
          Sæt status manuelt, hvis virkeligheden ikke passer med systemet — fx en klan der har
          betalt med bankoverførsel i stedet for MobilePay. Betalinger ovenfor ændres ikke.
        </p>
        <div class="flex flex-wrap items-center gap-2">
          <Select
            v-model="selectedStatus"
            :options="statusOptions"
            optionLabel="label"
            optionValue="slug"
            class="w-56"
            :disabled="busy"
          />
          <Button
            label="Gem status"
            icon="pi pi-check"
            size="small"
            :disabled="!statusDirty || busy"
            :loading="busy"
            @click="saveStatus"
          />
          <span v-if="statusDirty" class="text-sm italic text-amber-700">
            {{ statusLabel(team.status) }} → {{ statusLabel(selectedStatus ?? undefined) }}
          </span>
        </div>
      </Fieldset>
    </template>

    <template #footer>
      <!-- Left of everything, away from the ordinary way out of the dialog. -->
      <Button
        label="Slet klan"
        icon="pi pi-trash"
        severity="danger"
        text
        class="mr-auto"
        :disabled="busy || !detail"
        @click="remove"
      />
      <Button label="Luk" text @click="emit('close')" />
    </template>
  </Dialog>
</template>
