<script setup>
// The verification mark on a phone field: a green pi-verified seal when the spejder proved
// the number themselves in hej-appen (hej, PRD 015), an hourglass when they have not.
//
// The unverified state is drawn rather than left out on purpose. This is a checklist read
// under pressure at the counter, and the question is "which of these do I still have to
// ask about?" — so the marks must line up in a column the eye can run down. An addon that
// only appeared when verified would make every input a different width and turn that
// column into a jumble; worse, "no mark" and "no space for a mark" look identical, so a
// missing seal would read as a rendering slip rather than as an answer.
//
// The hourglass is kept very light: it means "still outstanding", which is the *ordinary*
// state before check-in, and a whole column of dark icons would read as a column of
// warnings. The eye should catch the green seals and pass over the rest.
defineProps({
  verified: { type: Boolean, default: false },
})
</script>

<template>
  <InputGroupAddon
    v-tooltip.top="verified ? 'Bekræftet af spejderen selv i hej-appen' : 'Ikke bekræftet — spørg ved check-in'"
    :aria-label="verified ? 'Bekræftet nummer' : 'Ikke bekræftet nummer'"
  >
    <i v-if="verified" class="pi pi-verified text-green-600" />
    <i v-else class="pi pi-hourglass text-gray-200" />
  </InputGroupAddon>
</template>
