<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, RouterView, useRoute } from 'vue-router'
import HelloWorld from './components/HelloWorld.vue'
import Navigation from '@/components/Navigation.vue'
import Foooter from '@/components/Footer.vue'
import TrackMapDialog from '@/components/TrackMapDialog.vue'
import { useToast } from 'primevue/usetoast'

const toast = useToast()
const route = useRoute()
const isFullbleed = computed(() => route.path === '/kort')
</script>

<template>
  <navigation title="Nathejk 2019" class="dark" />
  <main v-if="isFullbleed" role="main" class="fullbleed">
    <RouterView />
  </main>
  <template v-else>
    <main role="main" class="container mx-auto py5 maxw-screen-md">
      <RouterView />
    </main>
    <foooter />
  </template>
  <Toast />
  <!--
    Mounted once, here, rather than in each of the seven places a position glyph appears (PRD 011).
    Its content is owned by `useTrackViewer`, the same way Toast is driven by the toast service — seven
    copies would mean seven chances to wire it differently and a Leaflet map instantiated per list.
  -->
  <TrackMapDialog />
</template>

<style scoped>
.lightgrey {
  color: #d5d5d5;
}
.grey {
  color: #888;
}
.darkblue {
  color: #00008b;
}
.midnightblue {
  color: #445e65;
}
.hazyblue {
  color: #a2aeb2;
}

.bg-midnightblue {
  background-color: #445e65;
}
.bg-hazyblue {
  background-color: #a2aeb2;
}

.fullbleed {
  padding: 0;
  margin: 0;
  max-width: none;
  width: 100%;
  height: calc(100vh - 60px);
  overflow: hidden;
}
/*
header {
  line-height: 1.5;
  max-height: 100vh;
}

.logo {
  display: block;
  margin: 0 auto 2rem;
}

nav {
  width: 100%;
  font-size: 12px;
  text-align: center;
  margin-top: 2rem;
}

nav a.router-link-exact-active {
  color: var(--color-text);
}

nav a.router-link-exact-active:hover {
  background-color: transparent;
}

nav a {
  display: inline-block;
  padding: 0 1rem;
  border-left: 1px solid var(--color-border);
}

nav a:first-of-type {
  border: 0;
}

@media (min-width: 1024px) {
  header {
    display: flex;
    place-items: center;
    padding-right: calc(var(--section-gap) / 2);
  }

  .logo {
    margin: 0 2rem 0 0;
  }

  header .wrapper {
    display: flex;
    place-items: flex-start;
    flex-wrap: wrap;
  }

  nav {
    text-align: left;
    margin-left: -1rem;
    font-size: 1rem;

    padding: 1rem 0;
    margin-top: 1rem;
  }
}
*/
</style>
