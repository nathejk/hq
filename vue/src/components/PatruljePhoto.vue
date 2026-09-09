<script setup>
// A patrulje's photograph, wherever a patrulje is shown.
//
// One picture — the organizer's chosen cover, else the newest — from the single
// year-wide covers request, so dropping this into a 200-row table costs no extra
// requests. Clicking mounts the dialog, which is what loads that team's full list.
//
// Renders nothing at all when the team has no photograph. A placeholder in every
// row of a list during a race — when most teams have not been photographed yet —
// would be noise standing in for information.

import { computed, ref } from 'vue';
import { usePatruljeCovers } from '@/composables/patruljePhotos';
import PatruljePhotoDialog from '@/components/PatruljePhotoDialog.vue';

const props = defineProps({
  teamId: { type: String, required: true },
  teamName: { type: String, required: false, default: '' },
  // sm for table rows, md for an expanded row, lg for a patrol's own page.
  size: { type: String, required: false, default: 'sm' },
});

const { byTeam } = usePatruljeCovers();
const cover = computed(() => byTeam.value.get(props.teamId) ?? null);

const sizes = {
  sm: 'h-10 w-14',
  md: 'h-16 w-24',
  lg: 'h-28 w-40',
};
const boxClass = computed(() => sizes[props.size] ?? sizes.sm);

const open = ref(false);
</script>

<template>
  <button
    v-if="cover"
    type="button"
    class="relative shrink-0 overflow-hidden rounded border border-gray-300 hover:border-sky-500"
    :class="boxClass"
    :title="cover.count > 1 ? `${cover.count} billeder` : 'Se billede'"
    @click.stop="open = true"
  >
    <img
      :src="cover.thumbUrl || cover.url"
      :alt="teamName ? `Foto af ${teamName}` : 'Patruljefoto'"
      class="h-full w-full object-cover"
      loading="lazy"
      referrerpolicy="no-referrer"
    />
    <!-- How many there are, so an operator knows a click opens a carousel rather
         than one picture. -->
    <span
      v-if="cover.count > 1"
      class="absolute bottom-0 right-0 rounded-tl bg-black/60 px-1 text-[10px] leading-4 text-white"
    >
      {{ cover.count }}
    </span>
  </button>

  <PatruljePhotoDialog
    v-if="open"
    :teamId="teamId"
    :teamName="teamName"
    @close="open = false"
  />
</template>
