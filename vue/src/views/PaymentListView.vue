<script setup>
import { ref, onMounted } from 'vue'
import { http } from '@/plugins/axios'

onMounted(() => load())

const orders = ref([])
const expandedRows = ref([])
// orderId -> line items, lazily fetched on row expand (the list endpoint
// omits lines to stay light; the detail endpoint hydrates them).
const linesByOrder = ref({})

const load = async () => {
  try {
    const response = await http.get('/orders')
    orders.value = response.data.orders || []
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

const statusLabel = (status) => (status === 'PAID' ? 'Betalt' : 'Åben')
const statusSeverity = (status) => (status === 'PAID' ? 'success' : 'warn')
</script>

<template>
  <h1 class="font-nathejk text-2xl">Betalinger</h1>
  <div class="card" id="orders">
    <DataTable
      :value="orders"
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
      <template #loading>Henter bestillinger - vent... </template>
      <template #empty>Ingen bestillinger fundet</template>
      <Column expander />
      <Column field="createdAt" header="Tid" sortable>
        <template #body="{ data }">
          {{ formatDateTime(data.createdAt) }}
        </template>
      </Column>
      <Column field="ownerName" header="Ejer" sortable></Column>
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
      <Column field="status" header="Status">
        <template #body="{ data }">
          <Tag :value="statusLabel(data.status)" :severity="statusSeverity(data.status)" />
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
