<script setup>
import { ref, computed, watch } from 'vue'
import { useToast } from 'primevue/usetoast'
import { FilterMatchMode } from '@primevue/core/api'
import { http } from '@/plugins/axios'
import { useLiveResource } from '@/composables/useLiveResource'
import { daymonthhhmm } from '@/composables/datefilters'

const toast = useToast()

const expandedRows = ref([])

// Danish display label for an order's ownerType. Anything we don't recognise
// falls back to "Andet" so the Type column (and its filter) is exhaustive.
const ownerTypeLabel = (ownerType) => {
  switch (ownerType) {
    case 'patrulje':
      return 'Patrulje'
    case 'klan':
      return 'Klan'
    case 'gøgler':
      return 'Gøgler'
    case 'crew':
      return 'Crew'
    default:
      return 'Andet'
  }
}
const typeOptions = ['Patrulje', 'Klan', 'Gøgler', 'Crew', 'Andet']

// The API emits a binary status (OPEN/PAID; cancelled collapses into PAID).
const statusLabelOf = (status) => (status === 'PAID' ? 'Betalt' : 'Åben')
const statusOptions = ['Åben', 'Betalt']

// Filter on the Danish labels so "Andet" is filterable and the default status
// value can be expressed as "Betalt" directly.
const filters = ref({
  typeLabel: { value: null, matchMode: FilterMatchMode.EQUALS },
  statusLabel: { value: 'Betalt', matchMode: FilterMatchMode.EQUALS }
})

// Live, cached list. Returning to this page renders from cache with no request,
// and an order paid elsewhere appears without a refresh.
//
// dependsOn covers both entities behind a row: `order` for the order itself and
// its lines, `payment` because the OPEN/PAID status a row shows is settled by the
// payment projection, which is a separate event from the order's own.
const { data: ordersData, pending, error } = useLiveResource(
  'payment:list',
  async () => {
    const response = await http.get('/orders')
    return (response.data.orders || []).map((o) => ({
      ...o,
      typeLabel: ownerTypeLabel(o.ownerType),
      statusLabel: statusLabelOf(o.status)
    }))
  },
  { dependsOn: ['order', 'payment'] }
)

const orders = computed(() => ordersData.value ?? [])

watch(error, (err) => {
  if (!err) return
  console.log('orders list load failed', err)
  toast.add({ severity: 'error', summary: 'Kunne ikke hente ordrer', life: 5000 })
})

// The rows actually shown, i.e. after the Type/Status filters. Drives both the
// table and the summary so the two can never disagree. (The DataTable applies
// the same filters again, which is a no-op on an already-filtered list.)
const shownOrders = computed(() =>
  orders.value.filter((o) => {
    const type = filters.value.typeLabel.value
    const status = filters.value.statusLabel.value
    if (type && o.typeLabel !== type) return false
    if (status && o.statusLabel !== status) return false
    return true
  })
)

// Product variant (e.g. t-shirt size) is recorded on the line's attributes.
const lineSize = (line) => {
  const a = line.attributes || {}
  return a.size || a.tshirtSize || a.Size || null
}

// Aggregate line items across the shown orders. Merchandise with a size (e.g.
// t-shirts) is counted per size, so each size gets its own row.
const lineSummary = computed(() => {
  const counts = {}
  for (const order of shownOrders.value) {
    for (const line of order.lines || []) {
      const size = lineSize(line)
      const label = size ? `${line.productName} (${String(size).toUpperCase()})` : line.productName
      counts[label] = (counts[label] || 0) + (line.quantity || 0)
    }
  }
  return Object.entries(counts)
    .map(([label, count]) => ({ label, count }))
    .sort((a, b) => a.label.localeCompare(b.label, 'da-DK'))
})

const formatAmount = (value, currency) => {
  if (value == null) return ''
  return (value / 100).toLocaleString('da-DK', { style: 'currency', currency: currency || 'DKK' })
}

// Shared rather than parsed here: payment.createdAt arrives as Go's time.Time text
// form, which Safari refuses to parse. See parseApiDate.
const formatDateTime = (value) => daymonthhhmm(value)

const statusSeverity = (statusLabel) => (statusLabel === 'Betalt' ? 'success' : 'warn')
</script>

<template>
  <h1 class="font-nathejk text-2xl">Ordrehistorik</h1>

  <div class="my-3 rounded border border-gray-200 bg-gray-50 p-3 text-sm">
    <div class="font-semibold pb-1">{{ shownOrders.length }} ordrer vist</div>
    <div v-if="lineSummary.length" class="flex flex-wrap gap-x-5 gap-y-1">
      <span v-for="item in lineSummary" :key="item.label">
        {{ item.label }}: <span class="font-semibold">{{ item.count }}</span>
      </span>
    </div>
    <div v-else class="text-gray-500 italic">Ingen ordrelinjer</div>
  </div>

  <div class="card" id="orders">
    <DataTable
      :value="shownOrders"
      :loading="pending"
      v-model:filters="filters"
      filterDisplay="row"
      sortMode="single"
      sortField="createdAt"
      :sortOrder="-1"
      :stripedRows="true"
      v-model:expandedRows="expandedRows"
      paginator
      :rows="50"
      dataKey="orderId"
    >
      <template #loading>Henter ordrer - vent... </template>
      <template #empty>Ingen ordrer fundet</template>
      <Column expander />
      <Column field="createdAt" header="Tid" sortable>
        <template #body="{ data }">
          {{ formatDateTime(data.createdAt) }}
        </template>
      </Column>
      <Column field="ownerName" header="Ejer" sortable></Column>
      <Column field="typeLabel" header="Type" sortable :showFilterMenu="false">
        <template #body="{ data }">
          {{ data.typeLabel }}
        </template>
        <template #filter="{ filterModel, filterCallback }">
          <Select
            v-model="filterModel.value"
            :options="typeOptions"
            placeholder="Alle"
            showClear
            size="small"
            class="w-28"
            @change="filterCallback()"
          />
        </template>
      </Column>
      <Column field="totalAmount" header="Beløb" sortable>
        <template #body="{ data }">
          {{ formatAmount(data.totalAmount, data.currency) }}
        </template>
      </Column>
      <Column field="paidAmount" header="Betalt">
        <template #body="{ data }">
          {{ formatAmount(data.paidAmount, data.currency) }}
        </template>
      </Column>
      <Column field="dueAmount" header="Mangler">
        <template #body="{ data }">
          {{ formatAmount(data.dueAmount, data.currency) }}
        </template>
      </Column>
      <Column field="statusLabel" header="Status" :showFilterMenu="false">
        <template #body="{ data }">
          <Tag :value="data.statusLabel" :severity="statusSeverity(data.statusLabel)" />
        </template>
        <template #filter="{ filterModel, filterCallback }">
          <Select
            v-model="filterModel.value"
            :options="statusOptions"
            placeholder="Alle"
            showClear
            size="small"
            class="w-24"
            @change="filterCallback()"
          />
        </template>
      </Column>
      <template #expansion="{ data }">
        <div class="p-3">
          <h3 class="font-semibold mb-2">Ordrelinjer</h3>
          <DataTable :value="data.lines || []">
            <template #empty>Ingen linjer</template>
            <Column field="productName" header="Produkt"></Column>
            <Column header="Størrelse">
              <template #body="{ data: line }">
                {{ lineSize(line) ? String(lineSize(line)).toUpperCase() : '' }}
              </template>
            </Column>
            <Column field="memberId" header="Medlem"></Column>
            <Column field="quantity" header="Antal"></Column>
            <Column field="unitPrice" header="Stykpris">
              <template #body="{ data: line }">
                {{ formatAmount(line.unitPrice, data.currency) }}
              </template>
            </Column>
            <Column field="lineTotal" header="Linjetotal">
              <template #body="{ data: line }">
                {{ formatAmount(line.lineTotal, data.currency) }}
              </template>
            </Column>
          </DataTable>
        </div>
      </template>
    </DataTable>
  </div>
</template>

<style>
#orders td {
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
