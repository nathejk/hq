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

// The comment text is the only thing on this timeline an operator writes; the rest
// is the system recording what happened. That difference drives the styling: a
// comment gets a tinted card, a state change is a dimmed one-liner that can be
// skimmed past.
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
  <!-- A comment is somebody's own words, so it gets a card: it should read as
       content on the rail, clearly distinct from the one-line state changes around
       it. State changes stay plain text — a card each would turn the timeline back
       into a wall of boxes. -->
  <Card v-if="isComment" class="sos-comment border border-gray-200">
    <template #content>
      <div v-if="editing" class="flex flex-col gap-1">
        <Textarea v-model="draftValue" rows="2" class="w-full" autofocus />
        <div class="flex gap-1">
          <Button label="Gem" size="small" @click="emit('save')" />
          <Button label="Annuller" size="small" severity="secondary" text @click="emit('cancel')" />
        </div>
      </div>

      <div v-else class="flex items-start gap-2 group">
        <div class="flex-1">
          <span class="whitespace-pre-wrap">{{ commentText }}</span>
          <span v-if="edited" class="ml-2 text-xs italic text-gray-500">(redigeret)</span>
        </div>
        <Button class="comment-edit" icon="pi pi-pencil" text rounded size="small"
                @click="emit('edit')" />
      </div>
    </template>
  </Card>

  <div v-else class="text-sm text-gray-400">
    <span>{{ activityLabel(activity.type) }}</span>
    <span v-if="detail" class="text-gray-500">: {{ detail }}</span>
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

/* A card on a timeline rail wants to be a quiet container, not a raised panel:
   tighter than the default padding and flat rather than shadowed, so a run of
   comments does not become a stack of floating boxes.

   The tint matters more than it looks: the page's own cards are white, so a white
   comment card would dissolve into the panel behind it. A slightly cool grey makes
   each comment read as a note laid on the page. Set here rather than with a Tailwind
   class because PrimeVue's own `.p-card` background would otherwise win — a scoped
   rule carries the extra attribute selector and lands on top. */
.sos-comment {
  background: #f4f6f8;
  box-shadow: none;
}
.sos-comment :deep(.p-card-body) {
  padding: 0.6rem 0.75rem;
}
.sos-comment :deep(.p-card-content) {
  padding: 0;
}
</style>
