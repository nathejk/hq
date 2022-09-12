<template>
  <LMap ref="map"
    :zoom="zoom"
    :center="center"
    :options="mapOptions"
    @update:center="centerUpdate"
    @update:zoom="zoomUpdate"
    @click="addMarker"
  >

    <LTileLayer :url="url" />
    <LWMSTileLayer
      :key="layer.name"
      :base-url="baseUrl"
      :token="layer.token"
      :layers="layer.layers"
      :name="layer.name"
      layer-type="base"
      :transparent="true"
      :options="mapOptions"
      :attribution="layer.attribution"
    />
    <LMarker v-if="currentMarker" :draggable="true" :latLng="currentMarker" @update:latLng="updateMarker" />

  </LMap>
</template>

<style>

</style>
<script>
import { Icon, latLng } from "leaflet";
import { LMap, LMarker, LTileLayer, LWMSTileLayer } from 'vue2-leaflet';
import 'leaflet/dist/leaflet.css';

delete Icon.Default.prototype._getIconUrl;
Icon.Default.mergeOptions({
  iconRetinaUrl: require('leaflet/dist/images/marker-icon-2x.png'),
  iconUrl: require('leaflet/dist/images/marker-icon.png'),
  shadowUrl: require('leaflet/dist/images/marker-shadow.png'),
});
export default {
    props: {
      marker: null,
    },
    data: () => ({
      currentMarker: null,
      zoom: 13,
      center: latLng(55.2483, 11.5246),
      url: 'https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png',
      //attribution: '&copy; <a href="http://osm.org/copyright">OpenStreetMap</a> contributors',
      //url: '',
      baseUrl: 'https://kortforsyningen.kms.dk/topo50',
      layer: {
        layers: 'dtk_2cm_508dpi',
        format: 'image/png',
        attribution: '&copy; <a target="_blank" href="https://download.kortforsyningen.dk/content/vilk%C3%A5r-og-betingelser">Styrelsen for Dataforsyning og Effektivisering</a>',
        name: 'Topografisk 2-cm kort',
      },
      withTooltip: latLng(47.41422, -1.250482),
      currentZoom: 11.5,
      mapOptions: {
        zoomSnap: 0.5,
        token: '0d5816d7e175e934301f0277686c43f8',
      },
    }),
    components: { LMap, LMarker, LTileLayer, LWMSTileLayer },
    filters: {
    },
    watch: {
      currentMarker: function(newVal, oldVal) {
        this.$emit('marker:updated', newVal);
        console.log(newVal, oldVal)
      }
    },
    methods: {
        zoomUpdate(zoom) { this.currentZoom = zoom; },
        centerUpdate(center) { this.currentCenter = center; },
        addMarker(marker) { this.currentMarker = marker.latlng; },
        updateMarker(marker) { this.currentMarker = latLng(marker.lat, marker.lng)  },
    },
    mounted() {
        this.currentMarker = this.marker
    }
}
</script>
