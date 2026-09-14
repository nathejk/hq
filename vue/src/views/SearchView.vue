<script setup lang="ts">
// Person search results (PRD 014, task 182).
//
// # The query lives in the URL, and the URL is the cache key
//
// One value, three jobs: it makes a search linkable, it survives reload, and it is what
// `useLiveResource` is keyed by. The debounce sits *before* the URL is written, which is the point
// the task file stresses: every distinct key is a module-level cache entry that outlives this
// route, so feeding raw keystrokes into the key would leave an entry behind for every prefix of
// everything ever typed. `router.replace` rather than `push`, so the back button leaves the page
// instead of walking back through the operator's typing.
//
// # Re-keying a composable that takes a static key
//
// `useLiveResource` is called once per key by design. As the committed query changes we therefore
// create a new resource in a detached `effectScope` and stop the previous one; stopping disposes
// the computeds we no longer read, while the cache entry itself stays (it is a cache — going back
// to a query renders it instantly, which is the whole reason `pending` must not be re-raised).

import { computed, effectScope, onBeforeUnmount, ref, shallowRef, watch, type EffectScope } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { http } from '@/plugins/axios'
import { useLiveResource, type LiveResource } from '@/composables/useLiveResource'
import {
  MIN_NAME_LENGTH,
  MIN_PHONE_DIGITS,
  SEARCH_DEPENDS_ON,
  type PersonSearchResponse,
  type PersonRow,
  displayName,
  emptyState,
  isSearchable,
  kindLabel,
  phoneRoleLabel,
  resultRoute,
  teamLabel,
  toRows,
} from '@/composables/personSearch'

const route = useRoute()
const router = useRouter()

/** What is in the box right now. Not the cache key. */
const text = ref(String(route.query.q ?? ''))

/** What has been committed to the URL, and therefore searched. */
const committed = computed(() => String(route.query.q ?? ''))

const DEBOUNCE_MS = 250
let debounce: ReturnType<typeof setTimeout> | undefined

const commit = (value: string) => {
  if (value === committed.value) return
  void router.replace({ name: 'search', query: value ? { q: value } : {} })
}

watch(text, (value) => {
  if (debounce) clearTimeout(debounce)
  debounce = setTimeout(() => commit(value), DEBOUNCE_MS)
})

/** Enter should not make the operator wait out the debounce. */
const commitNow = () => {
  if (debounce) clearTimeout(debounce)
  commit(text.value)
}

// Arriving from elsewhere — the navigation search box, a shared link, the back button — must fill
// the box. Guarded on inequality so this and the debounce above cannot chase each other.
watch(committed, (value) => {
  if (value === text.value) return
  if (debounce) clearTimeout(debounce)
  text.value = value
})

onBeforeUnmount(() => {
  if (debounce) clearTimeout(debounce)
})

/**
 * The key, or '' when there is nothing worth asking the server for.
 *
 * The minimum length is enforced here rather than only on the server so the first keystroke does
 * not ask for every person in the event.
 */
const cacheKey = computed(() =>
  isSearchable(committed.value) ? `search:person:${committed.value}` : '',
)

// `shallowRef`, not `ref`: a deep ref wraps the object in a reactive proxy, which *unwraps* the
// refs inside it, so `resource.value.pending.value` would silently be `undefined` — a table that
// never shows loading and a `data` that never reads. Shallow keeps the composable's own refs intact.
const resource = shallowRef<LiveResource<PersonSearchResponse> | undefined>()
let scope: EffectScope | undefined

watch(
  cacheKey,
  (key) => {
    scope?.stop()
    scope = undefined
    resource.value = undefined
    if (!key) return

    const q = committed.value
    scope = effectScope(true)
    resource.value = scope.run(() =>
      useLiveResource<PersonSearchResponse>(
        key,
        async () => {
          const response = await http.get('/search/person', { params: { q } })
          return response.data.search as PersonSearchResponse
        },
        // Entity types, and exactly the eight the projection can emit — see SEARCH_DEPENDS_ON.
        { dependsOn: [...SEARCH_DEPENDS_ON] },
      ),
    )
  },
  { immediate: true },
)

onBeforeUnmount(() => scope?.stop())

const response = computed(() => resource.value?.data.value)
const rows = computed(() => toRows(response.value?.results ?? []))

/**
 * `pending` straight from the resource, wired to the table's `:loading` and nothing else.
 *
 * It is true only when there is no cached value, so a query the operator returns to renders from
 * memory without flashing. A spinner of our own would undo exactly that.
 */
const pending = computed(() => resource.value?.pending.value ?? false)
const error = computed(() => resource.value?.error.value)

const state = computed(() => emptyState(committed.value, response.value))

/**
 * Announced count, for the screen reader and for the operator's own confidence.
 *
 * `truncated` is stated here as well as in the table footer, because a capped list read as
 * complete is how an operator concludes somebody is not in the event.
 */
const announcement = computed(() => {
  switch (state.value) {
    case 'idle':
      return 'Intet søgt.'
    case 'tooShort':
      return 'For kort til at søge.'
    case 'noMatch':
      return 'Ingen match.'
    default: {
      const n = rows.value.length
      if (!response.value) return 'Søger …'
      return response.value.truncated
        ? `Viser de første ${n} match. Der er flere.`
        : `${n} ${n === 1 ? 'match' : 'match'}.`
    }
  }
})

const clear = () => {
  text.value = ''
  commitNow()
}

/**
 * The whole row is the link, not just the name.
 *
 * A row-click handler *and* a real anchor on the name, deliberately: the click target for a hurried
 * operator should be the whole row, but a row handler is invisible to the keyboard and to
 * open-in-new-tab, so the anchor stays for both. A row with nowhere to go does nothing rather than
 * navigating somewhere approximate.
 */
const openRow = (row: PersonRow) => {
  const to = resultRoute(row)
  if (to) void router.push(to)
}
</script>

<template>
  <div class="card" id="person-search">
    <div class="flex flex-wrap gap-2 items-center justify-between mb-4">
      <h1 class="font-nathejk text-2xl">Søg efter person</h1>
      <div class="flex gap-2 items-center">
        <IconField>
          <InputIcon><i class="pi pi-search" /></InputIcon>
          <InputText
            v-model="text"
            autofocus
            class="w-80"
            type="search"
            placeholder="Telefonnummer eller navn"
            aria-label="Søg efter telefonnummer eller navn"
            @keyup.enter="commitNow"
          />
        </IconField>
        <Button
          v-if="text"
          label="Ryd"
          severity="secondary"
          text
          @click="clear"
        />
      </div>
    </div>

    <!--
      One live region for the result count, so the answer is announced rather than only drawn.
      `aria-live="polite"` and not assertive: the operator is on the phone.
    -->
    <p class="text-sm text-gray-600 mb-2" role="status" aria-live="polite">{{ announcement }}</p>

    <Message v-if="error" severity="error" :closable="false" class="mb-3">
      Kunne ikke søge. Det, der står nedenfor, er måske ikke aktuelt.
    </Message>

    <!--
      Three empty states, deliberately worded differently: an empty box and an empty result are
      different answers, and confusing them is how an operator tells a caller their child is not
      in the event.
    -->
    <div v-if="state === 'idle'" class="py-8 text-center text-gray-500">
      <p class="text-lg">Skriv et telefonnummer eller et navn.</p>
      <p class="text-sm">Både spejderens eget og forælderens nummer bliver søgt.</p>
    </div>

    <div v-else-if="state === 'tooShort'" class="py-8 text-center text-gray-500">
      <p class="text-lg">Bliv ved — der er ikke søgt endnu.</p>
      <p class="text-sm">
        Mindst {{ MIN_NAME_LENGTH }} bogstaver eller {{ MIN_PHONE_DIGITS }} cifre.
      </p>
    </div>

    <div v-else-if="state === 'noMatch'" class="py-8 text-center text-gray-500">
      <p class="text-lg">Ingen match på “{{ committed }}”.</p>
      <p class="text-sm">Der blev søgt i indeværende år.</p>
    </div>

    <template v-else>
      <Message v-if="response?.truncated" severity="warn" :closable="false" class="mb-3">
        Der er flere match end de {{ rows.length }} viste. Indsnævr søgningen.
      </Message>

      <!--
        One table with subheaders rather than a table per group: it gives `pending` a single place
        to land, and keeps the whole result set in one tab order.
      -->
      <DataTable
        :value="rows"
        :loading="pending"
        dataKey="rowKey"
        rowGroupMode="subheader"
        groupRowsBy="group"
        sortField="order"
        :sortOrder="1"
        :stripedRows="true"
        selectionMode="single"
        @row-click="openRow($event.data)"
      >
        <template #empty>Ingen match</template>
        <template #groupheader="{ data }">
          <span class="font-nathejk text-lg">{{ data.group }}</span>
        </template>

        <Column header="Navn">
          <template #body="{ data }">
            <router-link
              v-if="resultRoute(data)"
              :to="resultRoute(data)!"
              class="font-semibold underline decoration-dotted"
            >
              {{ displayName(data) }}
            </router-link>
            <span v-else class="font-semibold">{{ displayName(data) }}</span>
            <span class="ml-2 text-xs text-gray-500">{{ kindLabel(data.kind) }}</span>
          </template>
        </Column>

        <Column header="Telefon">
          <template #body="{ data }">
            <template v-if="data.phone">
              <a :href="`tel:${data.phone}`" class="tabular-nums">{{ data.phone }}</a>
              <!--
                The annotation is not decoration: an operator who dials a parent's number
                believing it is the scout's opens the call with the wrong sentence.
              -->
              <span v-if="phoneRoleLabel(data.phoneRole)" class="ml-2 text-xs text-amber-700">
                {{ phoneRoleLabel(data.phoneRole) }}
              </span>
            </template>
            <span v-else class="text-sm italic text-gray-500">intet nummer</span>
          </template>
        </Column>

        <Column header="Hold">
          <template #body="{ data }">
            <span v-if="teamLabel(data)">{{ teamLabel(data) }}</span>
            <span v-else class="text-gray-400">—</span>
          </template>
        </Column>

        <!-- Status badges are task 183; plain text until then. -->
        <Column header="Status">
          <template #body="{ data }">
            <span v-if="data.removed" class="mr-2">udmeldt</span>
            <span v-if="data.status">{{ data.status }}</span>
            <span v-else-if="!data.removed" class="text-gray-400">ukendt</span>
          </template>
        </Column>
      </DataTable>
    </template>
  </div>
</template>

<style>
#person-search td {
  padding: 0.25rem 0.75rem;
}
/* The row is the link, so it should look like one. */
#person-search tbody tr {
  cursor: pointer;
}
</style>
