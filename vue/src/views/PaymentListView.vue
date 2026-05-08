<script setup>
import { ref, onMounted } from 'vue'
import { useToast } from 'primevue/usetoast'
import { http } from '@/plugins/axios'

const toast = useToast()

onMounted(() => load())

const payments = ref([])
const load = async () => {
  try {
    const response = await http.get('/payments')
    payments.value = response.data.payments
  } catch (error) {
    console.log('payments list load failed', error)
  }
}
const selectedValue = ref(null)
const expandedRows = ref([])
const onRowExpand = (event) => {
  console.log(expandedRows.value)
  //   toast.add({ severity: 'info', summary: 'Row Group Expanded', detail: 'Value: ' + event.data, life: 3000 });
}
const onRowCollapse = (event) => {
  //  toast.add({ severity: 'success', summary: 'Row Group Collapsed', detail: 'Value: ' + event.data, life: 3000 });
}

const formatAmount = (value, currency) => {
  if (value == null) return ''
  return (value / 100).toLocaleString('da-DK', { style: 'currency', currency: currency })
}

const formatDateTime = (value) => {
  if (!value) return ''
  const date = new Date(value)
  const day = date.getDate()
  const month = date.toLocaleString('da-DK', { month: 'short' })
  const time = date.toLocaleString('da-DK', { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })
  return `${day}. ${month} ${time}`
}

const getSeverity = (status) => {
  const mapping = {
    requested: 'secondary',
    reserved: 'info',
    received: 'success',
    refunded: 'warn',
    failed: 'danger'
  }
  return mapping[status]
}
</script>

<template>
  <h1 class="font-nathejk text-2xl">Betalinger</h1>
  <div class="card" id="badut">
    <DataTable :value="payments" sortMode="single" sortField="createdAt" :sortOrder="1" :stripedRows="true" v-model:expandedRows="expandedRows" paginator :rows="50" dataKey="reference" @rowExpand="onRowExpand" @rowCollapse="onRowCollapse">
      <template #loading>Henter betalinger - vent... </template>
      <Column expander />
      <Column field="createdAt" header="Tid">
        <template #body="{ data }">
          {{ formatDateTime(data.createdAt) }}
        </template>
      </Column>
      <Column field="orderType" header="Type"></Column>
      <Column field="method" header="Metode"></Column>
      <Column field="amount" header="Beløb">
        <template #body="{ data }">
          {{ formatAmount(data.amount, data.currency) }}
        </template>
      </Column>
      <Column field="status" header="Status">
        <template #body="{ data }">
          <Tag :value="data.status" :severity="getSeverity(data.status)" />
        </template>
      </Column>
      <Column field="reference" header="Reference"></Column>
      <template #expansion="{ data }">
        <div class="flex gap-4 items-stretch">
          <Fieldset legend="Events" class="flex-1">
            <Timeline :value="data.operations">
              <template #opposite="{ item }">
                <small class="text-surface-500 dark:text-surface-400">{{ formatDateTime(item.time) }}</small>
              </template>
              <template #content="{ item }">
                <Tag :value="item.type" :severity="getSeverity(item.type)" />
                {{ formatAmount(item.amount, data.currency) }}
              </template>
            </Timeline>
          </Fieldset>
          <Fieldset legend="Ordre" class="flex-1"> </Fieldset>
        </div>
        <div class="">
          {{ data }}
        </div>
      </template>
    </DataTable>
  </div>
</template>

<style>
#badut td {
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
