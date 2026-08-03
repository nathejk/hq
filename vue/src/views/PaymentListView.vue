<script setup>
import { ref, onMounted } from 'vue'
import { FilterMatchMode } from '@primevue/core/api'
import { http } from '@/plugins/axios'

onMounted(() => load())

const orders = ref([])
const expandedRows = ref([])
// orderId -> line items, lazily fetched on row expand (the list endpoint
// omits lines to stay light; the detail endpoint hydrates them).
const linesByOrder = ref({})

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

const load = async () => {
  try {
    const response = await http.get('/orders')
    orders.value = (response.data.orders || []).map((o) => ({
      ...o,
      typeLabel: ownerTypeLabel(o.ownerType),
      statusLabel: statusLabelOf(o.status)
    }))
  } catch (error) {
    console.log('orders list load failed', error)
  }
}

const onRowExpand = async (event) => {
  const order = event.data
  if (linesByOrder.value[order.orderId]) return
  try {
    const response = await http.get('/order/' + order.orderId)
    linesByOrder.value[order.orderId] = response.data.order.lines || []
  } catch (error) {
    console.log('order detail load failed', error)
    linesByOrder.value[order.orderId] = []
  }
}

const formatAmount = (value, currency) => {
  if (value == null) return ''
  return (value / 100).toLocaleString('da-DK', { style: 'currency', currency: currency || 'DKK' })
}

const formatDateTime = (value) => {
  if (!value) return ''
  const date = new Date(value)
  const day = date.getDate()
  const month = date.toLocaleString('da-DK', { month: 'short' })
  const time = date.toLocaleString('da-DK', { hour: '2-digit', minute: '2-digit', hour12: false })
  return `${day}. ${month} ${time}`
}

const statusSeverity = (statusLabel) => (statusLabel === 'Betalt' ? 'success' : 'warn')
</script>

<template>
  <h1 class="font-nathejk text-2xl">Ordrehistorik</h1>
  <div class="card" id="orders">
    <DataTable
      :value="orders"
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
      @rowExpand="onRowExpand"
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
            class="w-full"
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
            class="w-full"
            @change="filterCallback()"
          />
        </template>
      </Column>
      <template #expansion="{ data }">
        <div class="p-3">
          <h3 class="font-semibold mb-2">Ordrelinjer</h3>
          <DataTable :value="linesByOrder[data.orderId] || []">
            <template #empty>Ingen linjer</template>
            <Column field="productName" header="Produkt"></Column>
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
