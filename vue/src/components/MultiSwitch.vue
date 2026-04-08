<script setup lang="ts">
import { computed, ref, watch } from 'vue';

type OptionType = {
  name: string;
  shortName: string;
  value: number;
};

const props = withDefaults(
  defineProps<{
    options: OptionType[];
    type?: string;
    class?: string;
    readOnly?: boolean;
    placeholder?: string;
  }>(),
  {
    readOnly: false,
    class: '',
  },
);
const emit = defineEmits<{
  change: [action: OptionType];
}>();
const valueModel = defineModel<number>('value', { default: 0 });

const options = computed(() => props.options );

const setValue = (option: OptionType) => {
  if (props.readOnly) return
  valueModel.value = option.value;
  emit('change', option);
};
const activeLabel = computed(() => {
  return options.value.find((o) => o.value === valueModel.value)?.name ?? props.placeholder;
});
</script>

<template>
  <div class="multiswitch" :class="[props.class, { disabled: readOnly }]">
    <div class="multiswitch-over">
      <div class="multiswitch-options">
        <button v-for="option in options" :key="option.value" :class="['multiswitch-option', { 'option-selected': option.value === valueModel.value }]" @click.prevent="setValue(option)">
          <span class="option-label">{{ option.shortName }}</span>
        </button>
      </div>
    </div>
    <div class="multiswitch-out">
      <div class="multiswitch-current-value">
        <span>{{ activeLabel }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>

.disabled .multiswitch-options,
.multiswitch:hover .multiswitch-out,
.multiswitch .multiswitch-over  {
  display: none;
}

.multiswitch:hover .multiswitch-over {
  display: block;
}
.multiswitch-over, .multiswitch-out {
    height:100%;
}

.multiswitch-options {
  height:100%;
  display: grid;
  grid-auto-flow: column;
  gap: .5rem;
}
.multiswitch-option {
  transition: opacity linear 80ms;
  border-radius: .5rem;
}
.multiswitch-option:hover {
  background: hsl(52, 100%, 50%);
}
.option-selected {
  background: hsl(52, 100%, 50%, 0.6);
}

.multiswitch-current-value {
  transition: opacity linear 80ms, background linear 150ms;
  display: flex;
  justify-content: center;
  align-items: center;
  height:100%;
}

.disabled {
  @apply text-surface-500 pointer-events-none;
  background: hsl(var(--bg-color-hs), 90%);
}

.disabled .multiswitch-current-value {
  background: hsl(var(--bg-color-hs), 95%);
}

</style>
