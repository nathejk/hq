<script setup lang="ts">
// One control for "where?" — a checkpoint, a lok, HQ, or whatever the caller actually said.
//
// **Free text is not a fallback here.** "På Slangerupvej ved skovbrynet" is the normal way to
// describe where a scout is standing (PRD 009 §6), and a picker that only offered known locations
// would be worked around by typing the road name into the description, where nothing can read it
// as a place. So this is an autocomplete that suggests, rather than a select that constrains.
//
// The value carries kind + refId + **label**, and the label is a copy on purpose: a checkpoint
// renamed after the fact must not silently rewrite what the desk was told to do.

import { computed, ref, watch } from 'vue'
import { type Place, placeKindLabel } from '@/composables/dispatch'

export interface PlaceOption {
  kind: Place['kind']
  refId: string
  label: string
}

const props = defineProps<{
  modelValue: Place
  places: PlaceOption[]
  placeholder?: string
}>()

const emit = defineEmits<{ (e: 'update:modelValue', value: Place): void }>()

const text = ref(props.modelValue?.label ?? '')
const suggestions = ref<{ label: string; items: PlaceOption[] }[]>([])

// Keep the visible text in step with the value, so a kind change that re-defaults the places
// (see DispatchTaskDialog) is visible in the field rather than only in the payload.
watch(
  () => props.modelValue?.label,
  (label) => {
    if ((label ?? '') !== text.value) text.value = label ?? ''
  },
)

const groups = computed(() => {
  const order: Place['kind'][] = ['hq', 'checkpoint', 'lok']
  return order
    .map((kind) => ({
      label: placeKindLabel(kind),
      items: props.places.filter((p) => p.kind === kind),
    }))
    .filter((group) => group.items.length > 0)
})

function search(event: { query: string }) {
  const query = event.query.trim().toLowerCase()
  suggestions.value = groups.value
    .map((group) => ({
      label: group.label,
      items: query ? group.items.filter((i) => i.label.toLowerCase().includes(query)) : group.items,
    }))
    .filter((group) => group.items.length > 0)
}

function onSelect(event: { value: PlaceOption }) {
  const option = event.value
  text.value = option.label
  emit('update:modelValue', { kind: option.kind, refId: option.refId, label: option.label })
}

/**
 * Whatever was typed becomes a text place.
 *
 * Emitted on every change rather than only on blur: a dialog saved with Enter must not lose the
 * last thing typed, and "the value is what you can see" is the only rule an operator at 3am
 * should have to hold.
 */
function onInput(value: string | PlaceOption) {
  if (typeof value !== 'string') return
  emit('update:modelValue', { kind: 'text', refId: '', label: value })
}
</script>

<template>
  <AutoComplete
    v-model="text"
    :suggestions="suggestions"
    optionGroupLabel="label"
    optionGroupChildren="items"
    optionLabel="label"
    dropdown
    completeOnFocus
    :forceSelection="false"
    class="w-full"
    :placeholder="placeholder ?? 'Post, lok, HQ eller fri tekst'"
    @complete="search"
    @option-select="onSelect"
    @update:modelValue="onInput"
  >
    <template #optiongroup="{ option }">
      <span class="text-xs uppercase tracking-wide text-gray-500">{{ option.label }}</span>
    </template>
  </AutoComplete>
</template>
