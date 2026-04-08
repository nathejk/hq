<script setup lang="ts">
import { computed, ref, useId, watch } from 'vue'

const svgpath = {
  arrowup:
    'M12.2097 10.4113C12.1057 10.4118 12.0027 10.3915 11.9067 10.3516C11.8107 10.3118 11.7237 10.2532 11.6506 10.1792L6.93602 5.46461L2.22139 10.1476C2.07272 10.244 1.89599 10.2877 1.71953 10.2717C1.54307 10.2556 1.3771 10.1808 1.24822 10.0593C1.11933 9.93766 1.035 9.77633 1.00874 9.6011C0.982477 9.42587 1.0158 9.2469 1.10338 9.09287L6.37701 3.81923C6.52533 3.6711 6.72639 3.58789 6.93602 3.58789C7.14565 3.58789 7.3467 3.6711 7.49502 3.81923L12.7687 9.09287C12.9168 9.24119 13 9.44225 13 9.65187C13 9.8615 12.9168 10.0626 12.7687 10.2109C12.616 10.3487 12.4151 10.4207 12.2097 10.4113Z',
  arrowdown:
    'M7.01744 10.398C6.91269 10.3985 6.8089 10.378 6.71215 10.3379C6.61541 10.2977 6.52766 10.2386 6.45405 10.1641L1.13907 4.84913C1.03306 4.69404 0.985221 4.5065 1.00399 4.31958C1.02276 4.13266 1.10693 3.95838 1.24166 3.82747C1.37639 3.69655 1.55301 3.61742 1.74039 3.60402C1.92777 3.59062 2.11386 3.64382 2.26584 3.75424L7.01744 8.47394L11.769 3.75424C11.9189 3.65709 12.097 3.61306 12.2748 3.62921C12.4527 3.64535 12.6199 3.72073 12.7498 3.84328C12.8797 3.96582 12.9647 4.12842 12.9912 4.30502C13.0177 4.48162 12.9841 4.662 12.8958 4.81724L7.58083 10.1322C7.50996 10.2125 7.42344 10.2775 7.32656 10.3232C7.22968 10.3689 7.12449 10.3944 7.01744 10.398Z'
}
type OptionType = {
  name: string
  shortName: string
  value: int
}

const props = withDefaults(
  defineProps<{
    class?: string
    readOnly?: boolean
  }>(),
  {
    readOnly: false,
    class: ''
  }
)
const emit = defineEmits<{
  change: [action: OptionType]
}>()
const valueModel = defineModel<string>('value', { default: '00:00' })

const setValue = (h: int, m: int) => {
  valueModel.value = zeropad(h) + ':' + zeropad(m)
  emit('change', valueModel.value)
}
const h = computed(() => Number(valueModel.value.split(':')[0]))
const m = computed(() => Number(valueModel.value.split(':')[1]))

const hour = {
  plus: (x) => setValue((24 + h.value + x) % 24, m.value),
  minus: (x) => setValue((24 + h.value - x) % 24, m.value),
  set: (x) => setValue(x, m.value)
}
const minute = {
  plus: (x) => setValue(h.value, (60 + m.value + x) % 60),
  minus: (x) => setValue(h.value, (60 + m.value - x) % 60),
  set: (x) => setValue(h.value, x)
}
const popoverid = 'popover-' + useId()
const popover = {
  toggle: (opt) => {
    document.getElementById(popoverid).togglePopover(opt)
  },
  hours: (e) => {
    picks.value = [...Array(24).keys()]
    popover.pick = (x) => hour.set(x)
    popover.toggle({ force: true, source: e.target })
  },
  minutes: (e) => {
    picks.value = [
      ...Array(12)
        .keys()
        .map((v) => v * 5)
    ]
    popover.pick = (x) => minute.set(x)
    popover.toggle({ force: true, source: e.target })
  },
  close: (e) => popover.toggle({ force: false }),
  pick: () => 0
}
const picks = ref([])
const zeropad = (v) => String(v).padStart(2, '0')
</script>

<template>
  <div class="timepicker" :class="[props.class, { disabled: readOnly }]">
    <div class="timepicker-hours">
      <button type="button" @click="hour.plus(1)">
        <svg
          width="14"
          height="14"
          viewBox="0 0 14 14"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path :d="svgpath.arrowup" fill="currentColor"></path>
        </svg>
      </button>
      <div class="timepicker-hour" @click="popover.hours">{{ zeropad(h) }}</div>
      <button type="button" @click="hour.minus(1)">
        <svg
          width="14"
          height="14"
          viewBox="0 0 14 14"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path :d="svgpath.arrowdown" fill="currentColor"></path>
        </svg>
      </button>
    </div>
    <div class="timepicker-delimiter">
      <span>:</span>
    </div>
    <div class="timepicker-minutes">
      <button type="button" @click="minute.plus(1)">
        <svg
          width="14"
          height="14"
          viewBox="0 0 14 14"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path :d="svgpath.arrowup" fill="currentColor"></path>
        </svg>
      </button>
      <div class="timepicker-minute" @click="popover.minutes">{{ zeropad(m) }}</div>
      <button type="button" @click="minute.minus(1)">
        <svg
          width="14"
          height="14"
          viewBox="0 0 14 14"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path :d="svgpath.arrowdown" fill="currentColor"></path>
        </svg>
      </button>
    </div>
  </div>

  <div :id="popoverid" class="timepicker-popover" popover @click="popover.close">
    <div class="timepicker-picks">
      <div v-for="ph in picks" class="timepicker-pick" @click="popover.pick(ph)">
        {{ zeropad(ph) }}
      </div>
    </div>
  </div>
</template>

<style scoped>
.timepicker-popover {
  position: absolute;
  inset: auto auto 0 0;
  position-area: right center;
  background-color: rgba(240, 240, 240, 0.85);
  border: 1px solid #ddd;
  border-radius: 0.5rem;
  margin: 3px;
  padding: 0.75rem;
}
.timepicker-picks {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
}
.timepicker-pick {
  padding: 0.5rem;
  cursor: pointer;
}
.timepicker-pick:hover {
  border-radius: 50%;
  border: 1px solid #ddd;
  margin: -1px;
}

.timepicker {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 0.25rem;
}
.timepicker-hours,
.timepicker-minutes {
  display: flex;
  align-items: center;
  flex-direction: column;
  position: relative;
}
.timepicker-hour,
.timepicker-minute {
  cursor: pointer;
  padding: 0 0.5rem;
  user-select: none;
}
.timepicker-hour:hover,
.timepicker-minute:hover {
  color: #66f;
}

.timepicker button {
  color: #ccc;
  border-color: transparent;
  display: inline-flex;
  cursor: pointer;
  user-select: none;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  padding: 0 0.5rem;
}
.timepicker button:hover {
  color: #666;
}
</style>
