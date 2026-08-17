<script setup lang="ts">
import { computed } from 'vue'
import {
  activityLabel,
  severityLabel,
  isMemberSummaryType,
  parseMemberSummary,
  memberStatusPhrase,
} from '@/composables/sos'

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
// skimmed past. Only the *text* is dimmed — the marker and timestamp are the rail
// itself, and fading those makes the timeline harder to follow rather than quieter.
const isComment = computed(() => props.activity.type === 'commented')

// The member lifecycle summaries (PRD 006) carry JSON rather than a bare string, because
// one operation can touch several members and the line has to name them.
//
// Everything rendered comes from the stored summary — names, statuses, resulting strength
// — and nothing is looked up. That is the point: a member moved twice would otherwise have
// their *first* move described using their *second* team, and a timeline whose entries
// change meaning after the fact is worse than no timeline.
const memberSummary = computed(() =>
  isMemberSummaryType(props.activity.type) ? parseMemberSummary(props.activity.value) : null,
)

// Per-member phrases, shaped by what the operation was.
const memberLines = computed(() => {
  const s = memberSummary.value
  if (!s?.members?.length) return []
  return s.members.map((m) => {
    const who = m.name || m.memberId
    if (props.activity.type === 'member.moved') {
      return `${who} → ${m.toTeamName || m.toTeamId}`
    }
    // A status change reads as a transition, because "venter" alone does not say what
    // changed and a handover is read backwards from the latest entry.
    if (m.from && m.to) return `${who}: ${memberStatusPhrase(m.from)} → ${memberStatusPhrase(m.to)}`
    return who
  })
})

// The team's strength after the operation — the fact that makes a breach legible on the
// timeline rather than only in the live view. Zero is meaningful (the patrol is
// discontinued), so this checks for undefined rather than falsiness.
const strengthAfter = computed(() => {
  const s = memberSummary.value
  if (!s) return null
  const value = props.activity.type === 'member.moved' ? s.fromTeamStrength : s.teamStrength
  return value === undefined ? null : value
})

const summaryTeamName = computed(() => {
  const s = memberSummary.value
  if (!s) return ''
  return (props.activity.type === 'member.moved' ? s.fromTeamName : s.teamName) ?? ''
})

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
      // A member summary that failed to parse falls through to the raw value rather
      // than rendering nothing — unreadable beats absent on a handover log.
      return memberSummary.value ? '' : value
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
  <Card v-if="isComment" class="sos-note">
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
    <span v-if="summaryTeamName" class="text-gray-500">: {{ summaryTeamName }}</span>
    <span v-if="detail" class="text-gray-500">: {{ detail }}</span>

    <!--
      One entry, however many members the operation touched — which is the whole reason
      the backend publishes a single summarising event per operation rather than one per
      member. Rendered as a nested list so "hele patruljen hentes" reads as one thing that
      happened to four people, not four things.
    -->
    <ul v-if="memberLines.length" class="mt-0.5 ml-3 list-none text-gray-500">
      <li v-for="line in memberLines" :key="line">{{ line }}</li>
    </ul>
    <div v-if="strengthAfter !== null" class="ml-3 text-gray-500">
      <span v-if="strengthAfter === 0">Patruljen er udgået — ingen tilbage i løbet</span>
      <span v-else>{{ strengthAfter }} tilbage i løbet</span>
    </div>
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

/* The card's own surface — tint, border, flatness, padding — is the `.sos-note`
   class, defined in SosView's unscoped style block. It is shared with the case's own
   card deliberately: the two are required to look identical, and keeping one copy of
   the rules is the only way that survives the next adjustment. */
</style>
