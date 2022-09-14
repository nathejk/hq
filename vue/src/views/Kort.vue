<template>
    <div class="container-fluid h-100 p-0">
        <Map :center="latlng" :zoom="zoom"></Map>
    </div>
</template>

<style>
</style>

<script>
export default {
  components: {
    Map: () => import('@/components/Map.vue'),
  },
  data() {
    return {
      zoom: 12.5,
      latlng: "56.025652,12.314725",

    };
  },
  methods: {
  },
  mounted() {
  },
}
</script>
