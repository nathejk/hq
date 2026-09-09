<script setup>
// A patrulje's photographs, enlarged.
//
// Mounted only while open (the parent uses v-if), which is what keeps a 200-row
// list from fetching 200 photo lists: the per-team request happens on the click
// that opens this dialog. Reopening it is free — useLiveResource keeps the entry.
//
// Read-only except for one decision: which photograph represents the team. That is
// an event, so the button publishes and the live signal updates every open view;
// nothing is patched locally, so what you see is what was recorded.

import { computed, ref, watch } from 'vue';
import { useToast } from 'primevue/usetoast';
import {
  usePatruljePhotos,
  selectCoverPhoto,
  photoUrlFor,
} from '@/composables/patruljePhotos';
import { daymonthhhmm } from '@/composables/datefilters';

const props = defineProps({
  teamId: { type: String, required: true },
  // Only for the dialog's title. Optional: the map and the list both know it, the
  // scan trail does not, and a missing name is not worth a request.
  teamName: { type: String, required: false, default: '' },
});

const emit = defineEmits(['close']);

const toast = useToast();

const { data, pending } = usePatruljePhotos(props.teamId);
const photos = computed(() => data.value ?? []);

// Which photograph is on screen. Starts on the cover, because that is the one the
// operator clicked in the list — landing on a different picture would be
// disorienting.
const index = ref(0);
let followCover = true;
watch(
  photos,
  (list) => {
    if (!followCover || !list.length) return;
    const cover = list.findIndex((p) => p.cover);
    index.value = cover >= 0 ? cover : 0;
    followCover = false;
  },
  { immediate: true },
);

const current = computed(() => photos.value[index.value] ?? null);

// A screen-sized rendition rather than the ~2000px display image where one exists:
// see photoUrlFor. 1400 is generous for a dialog and still a fraction of the bytes.
const currentUrl = computed(() => (current.value ? photoUrlFor(current.value, 1400) : ''));

const step = (delta) => {
  if (photos.value.length < 2) return;
  const count = photos.value.length;
  index.value = (index.value + delta + count) % count;
};

const saving = ref(false);

// `ref` empty clears the choice. Both directions go through one function because
// both are the same event with a different value.
const choose = async (ref_) => {
  if (saving.value) return;
  saving.value = true;
  try {
    await selectCoverPhoto(props.teamId, ref_);
    toast.add({
      severity: 'success',
      summary: ref_ ? 'Forsidebillede valgt' : 'Forsidevalg fjernet',
      life: 3000,
    });
  } catch (error) {
    toast.add({
      severity: 'error',
      closable: true,
      life: 5000,
      summary: 'Kunne ikke gemme forsidebilledet',
      detail: error.message,
    });
  } finally {
    saving.value = false;
  }
};

const captured = (photo) => (photo?.capturedAt ? daymonthhhmm(photo.capturedAt) : '');
</script>

<template>
  <Dialog
    :visible="true"
    modal
    dismissableMask
    maximizable
    :style="{ width: '64rem', maxWidth: '95vw' }"
    @update:visible="emit('close')"
    @keydown.left="step(-1)"
    @keydown.right="step(1)"
  >
    <template #header>
      <div class="flex items-center gap-3">
        <span class="font-nathejk text-xl">{{ teamName || 'Patruljefoto' }}</span>
        <Tag v-if="photos.length > 1" :value="`${index + 1} / ${photos.length}`" severity="secondary" />
        <Tag v-if="current?.type" :value="current.type" severity="info" />
        <!-- The crew's "needs a look" flag (the camera app's XXX_ prefix). Carried
             through from the event rather than interpreted: what it means is the
             organizer's call. -->
        <Tag v-if="current?.attention" value="Se på denne" severity="warn" />
      </div>
    </template>

    <div v-if="pending" class="py-16 text-center text-gray-500">Henter billeder…</div>

    <div v-else-if="!photos.length" class="py-16 text-center text-gray-500">
      Der er ingen billeder af denne patrulje.
    </div>

    <div v-else>
      <div class="relative flex items-center justify-center bg-black/90">
        <img
          :src="currentUrl"
          :alt="teamName || 'Patruljefoto'"
          class="max-h-[70vh] w-auto max-w-full object-contain"
          referrerpolicy="no-referrer"
        />
        <template v-if="photos.length > 1">
          <Button
            icon="pi pi-chevron-left"
            rounded
            severity="secondary"
            class="!absolute left-2"
            aria-label="Forrige billede"
            @click="step(-1)"
          />
          <Button
            icon="pi pi-chevron-right"
            rounded
            severity="secondary"
            class="!absolute right-2"
            aria-label="Næste billede"
            @click="step(1)"
          />
        </template>
      </div>

      <!-- The picker. Only shown when there is a choice to make: with one
           photograph the cover is not a decision, and offering it would be noise. -->
      <div class="mt-3 flex flex-wrap items-center gap-2">
        <div v-if="photos.length > 1" class="flex flex-wrap gap-2">
          <button
            v-for="(photo, i) in photos"
            :key="photo.ref"
            type="button"
            class="relative rounded border-2 p-0"
            :class="i === index ? 'border-sky-500' : 'border-transparent'"
            :title="captured(photo)"
            @click="index = i"
          >
            <img
              :src="photo.thumbUrl || photo.url"
              alt=""
              class="h-16 w-24 rounded object-cover"
              referrerpolicy="no-referrer"
            />
            <i
              v-if="photo.cover"
              class="pi pi-star-fill absolute right-1 top-1 text-xs"
              :class="photo.chosen ? 'text-amber-400' : 'text-white/70'"
              :title="photo.chosen ? 'Valgt forsidebillede' : 'Nyeste billede'"
            />
          </button>
        </div>

        <div class="ml-auto flex items-center gap-2">
          <span v-if="captured(current)" class="text-sm text-gray-500">
            Taget {{ captured(current) }}
          </span>
          <Button
            v-if="photos.length > 1 && !(current?.cover && current?.chosen)"
            label="Vælg som forsidebillede"
            icon="pi pi-star"
            size="small"
            :loading="saving"
            @click="choose(current.ref)"
          />
          <Button
            v-else-if="current?.chosen"
            label="Fjern forsidevalg"
            icon="pi pi-star-fill"
            size="small"
            severity="secondary"
            outlined
            :loading="saving"
            @click="choose('')"
          />
        </div>
      </div>
    </div>
  </Dialog>
</template>
