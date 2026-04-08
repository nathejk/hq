<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()


const isOpen = ref(false)
const menuRef = ref<HTMLElement | null>(null)

function toggle() {
  isOpen.value = !isOpen.value
}

function close() {
  isOpen.value = false
}

function onClickOutside(e: MouseEvent) {
  if (!menuRef.value) return
  if (!menuRef.value.contains(e.target as Node)) {
    close()
  }
}

onMounted(() => {
  window.addEventListener('click', onClickOutside)
})

onBeforeUnmount(() => {
  window.removeEventListener('click', onClickOutside)
})

// emit actions to parent (logout, settings, etc.)
const emit = defineEmits<{
  (e: 'logout'): void
  (e: 'settings'): void
}>()

function goSettings() {
  emit('settings')
  close()
}

function logout() {
  emit('logout')
  close()
}

// extra internal links (same-site)
const links = ref([
  { label: 'Dashboard', to: { name: 'dashboard' } },
  { label: 'Alle årgange', to: { name: 'year' } },
  { label: 'Team', to: { name: 'team' } },
])

function goLink(link: { to: any }) {
  router.push(link.to)
  close()
}
</script>

<template>
  <div class="relative" ref="menuRef">
    <!-- Trigger -->
    <button
      type="button"
      @click.stop="toggle"
      class="flex items-center gap-2 rounded-full border border-gray-300 bg-white px-2 py-1 text-sm shadow-sm hover:bg-gray-50"
    >
      <span
        class="inline-flex h-8 w-8 items-center justify-center rounded-full bg-gray-200 text-xs font-medium text-gray-700"
      >
        NH
      </span>
      <span class="hidden text-sm font-medium text-gray-700 md:inline">
        Nathejk
      </span>
      <svg
        class="h-4 w-4 text-gray-500"
        fill="none"
        viewBox="0 0 20 20"
        stroke="currentColor"
      >
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
          d="M6 8l4 4 4-4" />
      </svg>
    </button>

    <!-- Dropdown -->
    <transition
      enter-active-class="transition ease-out duration-100"
      enter-from-class="transform opacity-0 scale-95"
      enter-to-class="transform opacity-100 scale-100"
      leave-active-class="transition ease-in duration-75"
      leave-from-class="transform opacity-100 scale-100"
      leave-to-class="transform opacity-0 scale-95"
    >
      <div
        v-if="isOpen"
        class="absolute right-0 mt-2 w-48 origin-top-right rounded-md bg-white py-1 shadow-lg ring-1 ring-black ring-opacity-5"
      >
        <!-- User info -->
        <div class="px-4 py-2 text-xs text-gray-500">
          Du er logget ind som<br />
          <span class="font-medium text-gray-800">nathejk</span>
        </div>

        <div class="my-1 border-t border-gray-100" />

        <!-- Links section -->
        <div class="px-2 py-1 text-xs font-semibold uppercase text-gray-400">
          Links
        </div>
        <button v-for="link in links" :key="link.label" type="button" @click="goLink(link)" class="block w-full px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-50">
          {{ link.label }}
        </button>

        <div class="my-1 border-t border-gray-100" />

        <!-- Account actions -->
        <button type="button" @click="goSettings" class="flex w-full items-center gap-2 px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-50">
            <!-- settings icon -->
          <svg class="h-4 w-4 text-gray-400" viewBox="0 0 20 20" fill="none" stroke="currentColor">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="1.5"
              d="M11.25 4.21l.44-1.32A1 1 0 0112.64 2h.72a1 1 0 01.95.68l.44 1.32a5.5 5.5 0 011.35.78l1.37-.46a1 1 0 011.18.45l.36.62a1 1 0 01-.23 1.27l-1.08.86c.04.26.07.53.07.8s-.03.54-.07.8l1.08.86a1 1 0 01.23 1.27l-.36.62a1 1 0 01-1.18.45l-1.37-.46a5.5 5.5 0 01-1.35.78l-.44 1.32a1 1 0 01-.95.68h-.72a1 1 0 01-.95-.68l-.44-1.32a5.5 5.5 0 01-1.35-.78l-1.37.46a1 1 0 01-1.18-.45l-.36-.62a1 1 0 01.23-1.27l1.08-.86A5.6 5.6 0 017 10c0-.27.03-.54.07-.8l-1.08-.86a1 1 0 01-.23-1.27l.36-.62a1 1 0 011.18-.45l1.37.46a5.5 5.5 0 011.35-.78z"
            />
            <circle cx="12" cy="10" r="2.25" />
          </svg>
          <span>Settings</span>
        </button>
        <button
          type="button"
          @click="logout"
          class="block w-full px-4 py-2 text-left text-sm text-red-600 hover:bg-red-50"
        >
          Logout
        </button>
      </div>
    </transition>
  </div>
</template>
