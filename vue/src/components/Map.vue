<template>
        <l-map
          class="h-100"
          ref="map"
          :zoom="zoom"
          :center="centerLatLng"
          :options="mapOptions"
          @update:center="centerUpdate"
          @update:zoom="zoomUpdate"
        >
      <l-control-layers position="topleft" />
      <l-control-scale position="bottomright" :imperial="false" :metric="true" />
      <l-tile-layer v-for="layer in layersTms"
        :url="layer.url"
        :key="layer.name"
        :name="layer.name"
        :attribution="layer.attribution"
        layer-type="base"
      />
      <l-wms-tile-layer v-for="layer in layersWms"
        :key="layer.name"
        :attribution="layer.attribution"
        :transparent="layer.transparent"
        :base-url="layer.url"
        :format="layer.format"
        :layers="layer.layers"
        :visible="layer.visible"
        :name="layer.name"
        layer-type="base"
      />

      <l-marker :lat-lng="withPopup">
        <l-popup>
          <div @click="innerClick">
            I am a popup
            <p v-show="showParagraph">
              Lorem ipsum dolor sit amet, consectetur adipiscing elit. Quisque
              sed pretium nisl, ut sagittis sapien. Sed vel sollicitudin nisi.
              Donec finibus semper metus id malesuada.
            </p>
          </div>
        </l-popup>
      </l-marker>
      <l-marker :lat-lng="withTooltip">
        <l-tooltip :options="{ permanent: true, interactive: true }">
          <div @click="innerClick">
            I am a tooltip
            <p v-show="showParagraph">
              Lorem ipsum dolor sit amet, consectetur adipiscing elit. Quisque
              sed pretium nisl, ut sagittis sapien. Sed vel sollicitudin nisi.
              Donec finibus semper metus id malesuada.
            </p>
          </div>
        </l-tooltip>
      </l-marker>
    </l-map>
</template>

<style>
.leaflet-touch .leaflet-control-layers-toggle  {
    width:30px;
    height:30px;
}
.leaflet-retina .leaflet-control-layers-toggle {
    background-size: 18px 18px;
}

</style>

<script>
import 'leaflet/dist/leaflet.css';

import { Icon, latLng } from 'leaflet';
import { LMap, LTileLayer, LWMSTileLayer, LControlLayers, LControlScale, LMarker, LPopup, LTooltip } from 'vue2-leaflet';
/*
delete Icon.Default.prototype._getIconUrl;
Icon.Default.mergeOptions({
  iconRetinaUrl: require('leaflet/dist/images/marker-icon-2x.png'),
  iconUrl: require('leaflet/dist/images/marker-icon.png'),
  shadowUrl: require('leaflet/dist/images/marker-shadow.png'),
});*/
export default {
  props: {
    center: {
      type: String,
      default: "56.025652,13.314725",
    },
    zoom: {
      type: Number,
      default: 11.5,
    },
  },
  components: {
    LMap,
    LTileLayer,
    "l-wms-tile-layer": LWMSTileLayer,
    LControlLayers,
    LControlScale,
    LMarker,
    LPopup,
    LTooltip,
  },
  data() {
    return {
      //center: latLng(56.025652, 12.314725),
      //url: 'https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png',
      //attribution: '&copy; <a href="http://osm.org/copyright">OpenStreetMap</a> contributors',

      layersTms: [
        {
          url : 'https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}',
          name: 'Luftfoto',
          attribution: 'Tiles &copy; Esri &mdash; Source: Esri, i-cubed, USDA, USGS, AEX, GeoEye, Getmapping, Aerogrid, IGN, IGP, UPR-EGP, and the GIS User Community',
        },
      ],
      layersWms: [
       /* {
          url: 'https://services.datafordeler.dk/DKtopokort/dtk_50/1.0.0/WMS?username=KJRCGJTSDZ&password=sepnup-Pocdu8-voksyb',
          name: 'Topografisk 1:50.000',
          visible: true,
          format: 'image/png',
          layers: 'dtk_2cm_508dpi',
          //layers: 'dtk_50',
          transparent: true,
          attribution: '&copy; <a target="_blank" href="https://download.kortforsyningen.dk/content/vilk%C3%A5r-og-betingelser">Styrelsen for Dataforsyning og Effektivisering</a>',
        },*/
        {
          url: 'https://api.dataforsyningen.dk/dtk_25_DAF?service=WMS&request=GetCapabilities&token=0d5816d7e175e934301f0277686c43f8',
          name: 'Topografisk 1:25.000',
          visible: true,
          format: 'image/png',
          //layers: 'dtk_2cm_508dpi',
          layers: 'DTK25',
          transparent: true,
          attribution: '&copy; <a target="_blank" href="https://download.kortforsyningen.dk/content/vilk%C3%A5r-og-betingelser">Styrelsen for Dataforsyning og Effektivisering</a>',
        }
      ],
      withPopup: latLng(47.41322, -1.219482),
      withTooltip: latLng(47.41422, -1.250482),
      currentZoom: 8,
      currentCenter: latLng(47.41322, -1.219482),
      showParagraph: false,
      mapOptions: {
        zoomSnap: 0.5
      },
    };
  },
  computed: {
    centerLatLng() {
            console.log(this.center, String(this.center))
        const latlng = String(this.center).split(',')
            console.log(latlng)
        return latLng(latlng[0], latlng[1])
    },
  },
  methods: {
    zoomUpdate(zoom) {
      this.currentZoom = zoom;
    },
    centerUpdate(center) {
      this.currentCenter = center;
    },
    showLongText() {
      this.showParagraph = !this.showParagraph;
    },
    innerClick() {
      alert("Click!");
    }
  },
  mounted() {
    this.$nextTick(() => {
      this.$refs.map.mapObject.invalidateSize()
    })
  },
}
</script>
