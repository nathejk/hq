<script setup lang="ts">
// The person-search entry point (PRD 014 §7, task 185).
//
// In the chrome rather than on a page of its own because the operator's hands are on the phone, not
// on the navigation: the target interaction for an inbound call is shortcut, eight digits, one
// click. A page you must first navigate to costs a decision the operator does not have spare.
//
// # Deliberately not a dropdown of live results
//
// It would compete with `SearchView` for the same job, and the results need room — the matched
// number, whose number it is, the team and two status badges. A navbar dropdown would drop most of
// that, which is how an operator ends up ringing a scout who went home last night. See the task
// file: adding one is a decision to take with the PRD, not here.
//
// # Why `/` and not Cmd/Ctrl+K
//
// Cmd/Ctrl+K is the fashionable choice and it shadows two things people rely on: Firefox focuses its
// search bar, Chrome focuses the omnibox in search mode. `/` shadows nothing in either browser, is
// the established in-app convention (GitHub, GitLab, Jira), and needs no modifier — which matters
// when the other hand is holding a phone.

import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()

const text = ref('')

/** The PrimeVue InputText wrapper; `$el` is the input itself. */
const field = ref<{ $el?: HTMLInputElement } | null>(null)

const inputEl = () => field.value?.$el ?? null

/**
 * Kept in step with the URL, so the box shows what is actually being searched.
 *
 * Without this, arriving on `/search?q=20309696` from a shared link would leave an empty box above a
 * page full of results — and the operator's next keystroke would search from scratch.
 */
watch(
  () => route.query.q,
  (q) => {
    text.value = typeof q === 'string' ? q : ''
  },
  { immediate: true },
)

const submit = () => {
  const q = text.value.trim()
  if (!q) {
    focusField()
    return
  }
  // `push` here, unlike the debounced `replace` inside SearchView: this *is* a navigation the
  // operator made, and Back should return them to the page they were on.
  void router.push({ name: 'search', query: { q } })
}

const focusField = () => {
  const el = inputEl()
  if (!el) return
  el.focus()
  el.select()
}

/**
 * Is the keystroke already going somewhere that wants it?
 *
 * The shortcut must not fire while the operator is writing a note in an SOS case, so anything
 * text-accepting is left alone — including `contenteditable`, which is what the Quill editor in the
 * mail view uses and which is not an `<input>`.
 */
const isTypingTarget = (target: EventTarget | null): boolean => {
  const el = target as HTMLElement | null
  if (!el) return false
  // `isContentEditable` first, then the attribute: the property is the correct check but jsdom does
  // not implement it, and `closest` also catches a keystroke landing on a child node inside the
  // editable region, which is the common case in Quill.
  if (el.isContentEditable) return true
  if (el.closest?.('[contenteditable="true"], [contenteditable=""]')) return true
  const tag = el.tagName
  return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT'
}

const onKeydown = (event: KeyboardEvent) => {
  if (event.key !== '/') return
  // A modified slash is somebody else's shortcut, not ours.
  if (event.ctrlKey || event.metaKey || event.altKey) return
  if (isTypingTarget(event.target)) return

  // Only now, once we know the keystroke is ours: swallowing it earlier would break Firefox's
  // quick-find for a user who never wanted our search.
  event.preventDefault()
  focusField()
}

onMounted(() => document.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => document.removeEventListener('keydown', onKeydown))

const hint = computed(() => 'Tryk / for at søge efter telefonnummer eller navn')
</script>

<template>
  <!--
    A real form with `role="search"`: Enter submits without a keydown handler, and the landmark makes
    the box reachable by a screen reader's landmark navigation rather than only by tabbing the whole
    icon bar. Accessibility is a requirement here, not a nicety — see the task file.
  -->
  <form class="flex items-center px-3" role="search" @submit.prevent="submit">
    <label for="nav-person-search" class="sr-only">Søg efter person</label>
    <IconField>
      <InputIcon><i class="pi pi-search" /></InputIcon>
      <InputText
        id="nav-person-search"
        ref="field"
        v-model="text"
        type="search"
        class="w-44 lg:w-56"
        size="small"
        placeholder="Søg person  /"
        :aria-label="hint"
        :title="hint"
        @keydown.esc="text = ''"
      />
    </IconField>
    <!-- Submit exists for pointer users and for forms submitted by assistive tech; the visual
         affordance is the placeholder, so it takes no width. -->
    <button type="submit" class="sr-only">Søg</button>
  </form>
</template>

<style scoped>
/*
 * The field sits *inside* the dark nav, so it cannot use PrimeVue's default light input styling —
 * a white box in a gray-800 bar reads as a hole punched through it.
 *
 * The three values below are Tailwind's gray-700, gray-600 and gray-500, chosen against the nav's
 * own gray-800 background and its existing hover/active greys (see Navigation.vue, which uses
 * gray-700 for hover and gray-600 for the active item). So the field is a step lighter than the
 * bar at rest, and clearly lighter once it has focus — without ever going white, which would
 * glare on a screen being read at 3am in a dark room.
 *
 * They are literals rather than Tailwind classes because PrimeVue's own `.p-inputtext` rules would
 * otherwise win, and `!important` on a utility class is worse than one documented block.
 *
 * Note the coupling: if the nav's `bg-gray-800` changes, these want revisiting. There is no
 * shared token for it today, so a comment is the honest mechanism.
 */
:deep(.p-inputtext) {
  --nav-search-idle: #374151; /* gray-700 — a step lighter than the bar */
  --nav-search-hover: #4b5563; /* gray-600 — matches the nav's active item */
  --nav-search-active: #6b7280; /* gray-500 — clearly lighter, still not white */

  background: var(--nav-search-idle);
  border-color: var(--nav-search-hover);
  color: #fff;
  transition:
    background-color 150ms ease,
    border-color 150ms ease;
}

:deep(.p-inputtext:hover:not(:focus)) {
  background: var(--nav-search-hover);
}

:deep(.p-inputtext:focus) {
  background: var(--nav-search-active);
  /* Light enough to read as focused on its own, so the ring is reinforcement rather than the
     only signal — which matters for anyone who cannot see the ring's colour. */
  border-color: #d1d5db;
  color: #fff;
}

/*
 * Placeholder and icon are held at gray-400/gray-300 rather than inheriting: white on gray-500
 * makes the placeholder look like typed text, and an operator glancing down needs to know at once
 * whether the box already holds a number.
 */
:deep(.p-inputtext::placeholder) {
  color: #9ca3af;
}

:deep(.p-inputtext:focus::placeholder) {
  color: #d1d5db;
}

:deep(.p-inputicon) {
  color: #9ca3af;
}

/* Tailwind's own sr-only, scoped, so this component does not depend on the plugin set being enabled. */
.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border-width: 0;
}
</style>
