<script setup lang="ts">
import { computed, ref, watch, onMounted } from 'vue'
import MultiSwitch from '@/components/MultiSwitch.vue'
import TimePicker from '@/components/TimePicker.vue'
import { parseOffset, dayIndex, composeDayTime, dayOptions } from '@/composables/daytime'

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

const day = ref(0)
const timestring = ref('00:00')
const zeropad = (v: number | string) => String(v).padStart(2, '0')
const hhmm = (d: Date) => zeropad(d.getHours()) + ':' + zeropad(d.getMinutes())

const offsetDate = computed(() => parseOffset(props.offset))

// when external value changes, update local UI state
const updateFromModel = (ts: Date) => {
  timestring.value = hhmm(ts)
  day.value = dayIndex(ts, offsetDate.value, props.dayCount)
}
onMounted(() => {
  if (!valueModel.value) {
    const fallback = parseOffset(props.offset)
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
  const base = composeDayTime(offsetDate.value, dayVal, hh, mm)

  // emit only if actually changed to avoid redundant updates
  if (!valueModel.value || base.getTime() !== valueModel.value.getTime()) {
    valueModel.value = base
    emit('change', base)
  }
})
const days = computed(() => dayOptions(offsetDate.value, props.dayCount))
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
