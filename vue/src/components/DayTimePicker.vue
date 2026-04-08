<script setup lang="ts">
import { computed, ref, watch, onMounted } from 'vue'
import MultiSwitch from '@/components/MultiSwitch.vue'
import TimePicker from '@/components/TimePicker.vue'

const props = withDefaults(
  defineProps<{
    offset?: string
    dayCount?: number
    class?: string
    readOnly?: boolean
  }>(),
  {
    dayCount: 3,
    readOnly: false,
    class: ''
  }
)
const valueModel = defineModel<Date | null>('value', { default: null })

const emit = defineEmits<{
  change: [action: Date]
}>()

const day = ref(1)
const timestring = ref('00:00')
const zeropad = (v: number | string) => String(v).padStart(2, '0')
const hhmm = (d: Date) => zeropad(d.getHours()) + ':' + zeropad(d.getMinutes())

const offsetDate = computed(() => (props.offset ? new Date(props.offset) : new Date(0)))

// when external value changes, update local UI state
const updateFromModel = (ts: Date) => {
  const ms = Math.max(0, ts.getTime() - offsetDate.value.getTime())

  timestring.value = hhmm(ts)
  day.value = Math.floor(ms / (1000 * 3600 * 24))
}
onMounted(() => {
  if (!valueModel.value) {
    const fallback = new Date(props.offset ?? 0)
    valueModel.value = fallback
    updateFromModel(fallback)
  } else {
    updateFromModel(valueModel.value)
  }
})

watch(
  () => valueModel.value,
  (cur) => {
    if (!cur) return
    updateFromModel(cur)
  }
)

// when local UI state changes, emit a *new* Date, but don’t loop back
watch([day, timestring], ([dayVal, timeStr]) => {
  if (!timeStr) return
  const [hh, mm] = timeStr.split(':').map(Number)
  const base = new Date()
  base.setTime(offsetDate.value.getTime() + dayVal * 1000 * 3600 * 24)
  base.setHours(hh)
  base.setMinutes(mm)

  // emit only if actually changed to avoid redundant updates
  if (!valueModel.value || base.getTime() !== valueModel.value.getTime()) {
    valueModel.value = base
    emit('change', base)
  }
})
const week = [
  { name: 'Søndag', shortName: 'søn', value: 6 },
  { name: 'Mandag', shortName: 'man', value: 0 },
  { name: 'Tirsdag', shortName: 'tirs', value: 1 },
  { name: 'Onsdag', shortName: 'ons', value: 2 },
  { name: 'Torsdag', shortName: 'tors', value: 3 },
  { name: 'Fredag', shortName: 'fre', value: 4 },
  { name: 'Lørdag', shortName: 'lør', value: 5 }
]
const days = computed(() => {
  const d = new Date(props.offset ?? offsetDate.value)
  const out: { name: string; shortName: string; value: number }[] = []
  for (let i = 0; i < (props.dayCount ?? 0); i++) {
    const w = week[(d.getDay() + i) % 7]
    out.push({
      name: w.name,
      shortName: w.shortName,
      value: i
    })
  }
  return out
})
</script>

<template>
  <div class="daytimepicker" :class="[props.class, { disabled: readOnly }]">
    <MultiSwitch class="py-1 my-4 w-32 border-y-2" :options="days" v-model:value="day" />
    <TimePicker v-model:value="timestring" />
  </div>
</template>

<style scoped>
.daytimepicker {
  display: flex;
  gap: 0.25rem;
}
</style>
