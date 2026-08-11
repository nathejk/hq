<script setup lang="ts">
import { computed } from 'vue'
import { activityIcon, activityLabel, formatTime, severityLabel } from '@/composables/sos'

// One entry on a case's timeline.
//
// The component renders **unknown types gracefully** rather than switching
// exhaustively: PRD 006 adds member transitions, whole-team collection and
// understrength exceptions to this same timeline, and an entry an operator cannot
// fully interpret is still far better than one silently missing from a handover
// record.
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

// What the entry's value means depends on the type, so it is rendered rather than
// dumped: a severity is a Danish label, a team id is not worth showing at all
// (the team card lists the patrols), and a headline change shows the new headline.
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
  <div class="flex gap-2 items-start py-1 border-b border-surface-200 last:border-0">
    <i :class="activityIcon(activity.type)" class="mt-1 text-surface-500" />
    <div class="flex-1">
      <div class="text-xs text-surface-500">
        {{ formatTime(activity.createdAt) }} — {{ activityLabel(activity.type) }}
        <span v-if="edited" class="italic">(redigeret)</span>
      </div>

      <div v-if="isComment && editing" class="flex flex-col gap-1 mt-1">
        <Textarea v-model="draftValue" rows="2" class="w-full" />
        <div class="flex gap-1">
          <Button label="Gem" size="small" @click="emit('save')" />
          <Button label="Annuller" size="small" severity="secondary" text @click="emit('cancel')" />
        </div>
      </div>

      <div v-else-if="isComment" class="flex items-start gap-2">
        <span class="whitespace-pre-wrap flex-1">{{ commentText }}</span>
        <Button icon="pi pi-pencil" text rounded size="small" @click="emit('edit')" />
      </div>

      <div v-else-if="detail" class="whitespace-pre-wrap">{{ detail }}</div>
    </div>
  </div>
</template>
