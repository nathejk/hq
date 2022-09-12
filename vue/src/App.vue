<template>
  <div class="d-flex flex-column h-100">
    <navigation title="Nathejk 2019"/>
    <main role="main" class="flex-fill">
      <router-view></router-view>
    </main>
    <foooter />
  </div>
</template>

<style lang="scss">
/*
@import "~bootstrap/scss/bootstrap";
@import '~bootstrap-vue/src/index.css';
*/
html, body { height:100%; background-color:#f8f8f8;}

.lightgrey { color: #d5d5d5; }
.grey { color: #888; }
.darkblue { color:#00008B; }
.midnightblue { color:#445e65; } 
.hazyblue { color:#a2aeb2; }

.bg-midnightblue { background-color:#445e65; }
.bg-hazyblue { background-color:#a2aeb2; }
</style>

<script>
  import Navigation from '@/components/Navigation.vue'
  import Foooter from '@/components/Footer.vue'

  export default {
    components: { Navigation, Foooter },
  }
</script>
