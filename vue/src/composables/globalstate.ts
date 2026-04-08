import { ref, computed } from 'vue'

const navTitle = computed(() => 'Nathejk ' + yearSlug.value);
const yearSlug = computed(() => (new Date().getFullYear() != rawYearSlug.value) ? rawYearSlug.value : '');

const extraHeader = ref<string | null>(null)
const rawYearSlug = ref('')

export function useGlobalState() {
  const setNav = (title: string, header: string | null) => {
    navTitle.value = title
    extraHeader.value = header
  }
  const setYearSlug = (slug: string) => {
    rawYearSlug.value = slug
  }

  return { navTitle, extraHeader, setYearSlug, yearSlug }
}
