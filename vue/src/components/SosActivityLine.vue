<script setup lang="ts">
import { computed } from 'vue'
import { activityLabel, severityLabel } from '@/composables/sos'

// The content of one timeline entry.
//
// Icon and timestamp are *not* here: they are the PrimeVue Timeline's marker and
// opposite slots in SosView, so the rail stays aligned across entries of different
// heights. This component owns what the entry says, and the inline comment editor.
//
// It renders **unknown types gracefully** rather than switching exhaustively: PRD
// 006 adds member transitions, whole-team collection and understrength exceptions
// to this same timeline, and an entry an operator cannot fully interpret is far
// better than one silently missing from a handover record.
const props = defineProps<{
  activity: {
    seq: number
    type: string
    actorUserId: string
    activityId?: string
    refActivityId?: string
    value?: string
    createdAt: string
  }
  commentText?: string
  edited?: boolean
  editing?: boolean
  draft?: string
}>()

const emit = defineEmits<{
  edit: []
  save: []
  cancel: []
  'update:draft': [value: string]
}>()

const isComment = computed(() => props.activity.type === 'commented')

// What an entry's value means depends on its type, so it is rendered rather than
// dumped: a severity becomes its Danish label, and a team id is not shown at all
// because the patrol is listed by name in the team card beside it.
const detail = computed(() => {
  const { type, value } = props.activity
  if (!value) return ''
  switch (type) {
    case 'severity.specified':
      return severityLabel(value)
    case 'team.associated':
    case 'team.disassociated':
      return ''
    default:
      return value
  }
})

const draftValue = computed({
  get: () => props.draft ?? '',
  set: (value: string) => emit('update:draft', value),
})
</script>

<template>
  <!-- A comment is the operator's own words, so it is set in normal body text with
       its label above it. Everything else is a state change: the label carries the
       meaning and the detail is secondary. -->
  <div v-if="isComment">
    <div class="text-xs text-surface-500">
      {{ activityLabel(activity.type) }}
      <span v-if="edited" class="italic">(redigeret)</span>
    </div>

    <div v-if="editing" class="flex flex-col gap-1 mt-1">
      <Textarea v-model="draftValue" rows="2" class="w-full" autofocus />
      <div class="flex gap-1">
        <Button label="Gem" size="small" @click="emit('save')" />
        <Button label="Annuller" size="small" severity="secondary" text @click="emit('cancel')" />
      </div>
    </div>

    <div v-else class="flex items-start gap-2 group">
      <span class="whitespace-pre-wrap flex-1">{{ commentText }}</span>
      <Button class="comment-edit" icon="pi pi-pencil" text rounded size="small"
              @click="emit('edit')" />
    </div>
  </div>

  <div v-else class="text-surface-700">
    <span>{{ activityLabel(activity.type) }}</span>
    <span v-if="detail" class="font-medium">: {{ detail }}</span>
  </div>
</template>

<style scoped>
/* The pencil is one of many on this page; show it on the entry being pointed at
   rather than on all of them at once. */
.comment-edit {
  opacity: 0;
  transition: opacity 120ms ease;
}
.group:hover .comment-edit,
.group:focus-within .comment-edit {
  opacity: 1;
}
</style>
