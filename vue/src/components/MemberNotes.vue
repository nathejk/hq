<script setup lang="ts">
// The note trail about one scout (PRD 008): what was agreed with a guardian, what was said on the
// phone, what the next shift needs to know.
//
// **Host-agnostic on purpose.** It takes a memberId and knows nothing about where it is rendered.
// That is what makes PRD 008's open question — modal or expanded row — cheap and reversible: the
// same component goes in the member dialog today, and in a DataTable row expander tomorrow if the
// crew prefers it, with no second implementation and no second set of bugs.

import { computed, ref, watch } from 'vue'
import { useToast } from 'primevue/usetoast'
import { http } from '@/plugins/axios'
import { useLiveResource } from '@/composables/useLiveResource'
import { formatTimestamp, useNow } from '@/composables/shelter'

const props = defineProps<{
  memberId: string
  /** Shown above the form, e.g. the scout's name. Optional: the host usually says it already. */
  heading?: string
}>()

// Emitted whenever unsaved text appears or disappears, so the host can defer live updates and say
// so on screen. The component does not defer anything itself: it does not own the payload the host
// is rendering, and a component that reached out to pause somebody else's data would be the kind of
// coupling this file exists to avoid.
const emit = defineEmits<{ dirty: [boolean] }>()

const toast = useToast()
const now = useNow()

interface Note {
  noteId: string
  memberId: string
  note: string
  actorUserId: string
  createdAt: string
  updatedAt: string
}

// Live, and type-level on `spejder`: a note written by another crew member has an id this client has
// never seen, so an instance-keyed dependency could not make it appear. The note events are spejder
// subjects (PRD 008 §8), which is why no new live token was needed for any of this.
const { data, pending, error, refresh } = useLiveResource(
  `member:notes:${props.memberId}`,
  async () => {
    const response = await http.get(`/member/${props.memberId}/notes`)
    return (response.data.notes ?? []) as Note[]
  },
  { dependsOn: ['spejder'] },
)

const notes = computed<Note[]>(() => data.value ?? [])

/** Text in the add form. */
const draft = ref('')

/** The note being corrected, and its working text. */
const editing = ref<{ noteId: string; value: string } | null>(null)

/** A request is in flight; buttons are disabled and the form must not be cleared under it. */
const saving = ref(false)

const MAX = 2000

// Unsaved text is either a new note or a correction. Reported to the host rather than handled here,
// as described above.
const dirty = computed(() => draft.value.trim().length > 0 || editing.value !== null)
watch(dirty, (value) => emit('dirty', value), { immediate: true })

const remaining = computed(() => MAX - (editing.value?.value ?? draft.value).length)

// Only shown as it starts to matter. A counter that sits at "1974 tegn tilbage" all night is noise;
// one that appears at 200 is information.
const showRemaining = computed(() => remaining.value <= 200)

const edited = (note: Note) => note.updatedAt > note.createdAt

/**
 * The server's Danish message, or a fallback.
 *
 * The API answers `{error: {field: "message"}}` for validation and `{error: "message"}` otherwise,
 * and those strings are written for the crew — better than anything this component could invent.
 */
const errorMessage = (err: unknown, fallback: string): string => {
  const payload = (err as { response?: { data?: { error?: unknown } } })?.response?.data?.error
  if (typeof payload === 'string') return payload
  if (payload && typeof payload === 'object') {
    const first = Object.values(payload as Record<string, unknown>)[0]
    if (typeof first === 'string') return first
  }
  return fallback
}

const add = async () => {
  const text = draft.value.trim()
  if (!text || saving.value) return
  saving.value = true
  try {
    await http.post(`/member/${props.memberId}/notes`, { note: text })
    // Cleared only after the request succeeds. Clearing optimistically would lose a crew member's
    // paragraph the one time the network drops — and this is the screen where the paragraph took a
    // phone call to obtain.
    draft.value = ''
    await refresh()
  } catch (err) {
    toast.add({
      severity: 'error',
      closable: true,
      life: 8000,
      summary: 'Noten blev ikke gemt',
      detail: errorMessage(err, 'Prøv igen'),
    })
  } finally {
    saving.value = false
  }
}

const startEditing = (note: Note) => {
  editing.value = { noteId: note.noteId, value: note.note }
}

const cancelEditing = () => {
  editing.value = null
}

const saveEdit = async () => {
  const current = editing.value
  if (!current || saving.value) return
  const text = current.value.trim()
  if (!text) {
    toast.add({
      severity: 'warn',
      life: 5000,
      summary: 'Noten kan ikke være tom',
      detail: 'Skriv rettelsen, eller fortryd',
    })
    return
  }
  saving.value = true
  try {
    await http.patch(`/member/${props.memberId}/notes/${current.noteId}`, { note: text })
    editing.value = null
    await refresh()
  } catch (err) {
    toast.add({
      severity: 'error',
      closable: true,
      life: 8000,
      summary: 'Rettelsen blev ikke gemt',
      detail: errorMessage(err, 'Prøv igen'),
    })
  } finally {
    saving.value = false
  }
}

watch(error, (err) => {
  if (!err) return
  toast.add({ severity: 'error', summary: 'Kunne ikke hente noter', life: 5000 })
})
</script>

<template>
  <section class="flex flex-col gap-3">
    <h3 v-if="heading" class="font-nathejk text-lg">{{ heading }}</h3>

    <!-- Only while nothing is cached; a reopened scout must not flash. -->
    <p v-if="pending && !notes.length" class="text-sm italic text-gray-500">Henter noter…</p>

    <p v-else-if="!notes.length" class="text-sm italic text-gray-500">
      Ingen noter endnu. Skriv hvad der er aftalt, så næste vagt ved det.
    </p>

    <!--
      Oldest first: a trail is a story and reads in the order it happened. The list on the shelter
      screen shows the newest as a snippet, which is a different question.
    -->
    <ol v-else class="flex flex-col gap-2">
      <li
        v-for="note in notes"
        :key="note.noteId"
        class="rounded border border-gray-200 bg-gray-50 px-3 py-2"
      >
        <div v-if="editing?.noteId === note.noteId" class="flex flex-col gap-2">
          <Textarea v-model="editing.value" rows="3" autoResize :maxlength="MAX" class="w-full" />
          <div class="flex items-center gap-2">
            <Button
              label="Gem rettelse"
              icon="pi pi-check"
              size="small"
              :loading="saving"
              @click="saveEdit()"
            />
            <Button label="Fortryd" size="small" text @click="cancelEditing()" />
            <span v-if="showRemaining" class="text-xs text-gray-500">{{ remaining }} tegn tilbage</span>
          </div>
        </div>

        <template v-else>
          <p class="whitespace-pre-wrap">{{ note.note }}</p>
          <div class="mt-1 flex flex-wrap items-center gap-2 text-xs text-gray-500">
            <span>{{ formatTimestamp(note.createdAt, now) }}</span>
            <!--
              Only when it differs, and worded as a correction. Nothing says who: until HQ has login
              every note is unsigned, and "Ukendt" would read as data loss rather than as a system
              that does not know yet (PRD 008 §5).
            -->
            <span v-if="edited(note)">· rettet {{ formatTimestamp(note.updatedAt, now) }}</span>
            <span v-if="note.actorUserId">· {{ note.actorUserId }}</span>
            <!--
              "Ret" rather than "Rediger", and deliberately understated: editing is for typos. The
              way to add information is a new note, which is why the form below is the prominent
              thing on this panel.
            -->
            <Button
              label="Ret"
              size="small"
              text
              class="!p-0 !text-xs"
              @click="startEditing(note)"
            />
          </div>
        </template>
      </li>
    </ol>

    <!-- The primary action: writing a new note, not correcting an old one. -->
    <div class="flex flex-col gap-2 border-t border-gray-200 pt-3">
      <Textarea
        v-model="draft"
        rows="3"
        autoResize
        :maxlength="MAX"
        class="w-full"
        placeholder="Fx: Ringet til mor 01.20. Hun henter kl. 06. Må sove i hallen."
      />
      <div class="flex items-center gap-2">
        <Button
          label="Tilføj note"
          icon="pi pi-plus"
          size="small"
          :disabled="!draft.trim()"
          :loading="saving"
          @click="add()"
        />
        <span v-if="showRemaining" class="text-xs text-gray-500">{{ remaining }} tegn tilbage</span>
      </div>
    </div>
  </section>
</template>
