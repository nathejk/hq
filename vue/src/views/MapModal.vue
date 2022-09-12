<template>
  <div ref="modal" id="mapModal" class="modal fade" tabindex="-1" role="dialog" aria-labelledby="mapModalTitle" aria-hidden="true">
    <div class="modal-dialog modal-dialog-centered modal-lg" role="document">
      <div class="modal-content">
        <div class="modal-header">
          <h6 class="modal-title" id="mapModalTitle">
            Angiv placering på kortet
            <!--br />
            <small class="small grey">location.display_name </small-->
          </h6>
          <button type="button" class="close" data-dismiss="modal" aria-label="Close">
            <span aria-hidden="true">
              <i class="fas fa-times-circle"></i>
            </span>
          </button>
        </div>
        <div class="modal-body p-0">
          <div style="height:466px">
            <TopoMap :key="renderModal"
              :zoom="13"
              :center="null"
              :marker="selectedPosition"
              @marker:updated="updatePosition"
            ></TopoMap>
          </div>
        </div>
        <div class="modal-footer py-1">
          <button type="button" class="btn btn-outline-secondary" data-dismiss="modal">Afbryd</button>
          <button type="button" class="btn btn-info" data-dismiss="modal" :disabled="!selectedPosition" @click="acceptPosition">OK</button>
      </div>
      </div>
    </div>
  </div>
</template>

<script>
import TopoMap from '@/components/TopoMap'

export default {
  data: () => ({
    selectedPosition: null,
    renderModal: 0,
  }),
  props: {
    position: null,
  },
  components: { TopoMap },
  computed: {
  },
  mounted() {
          /*
    $('#mapModal').on('shown.bs.modal', (e) => {
      this.renderModal += 1
    })*/
    this.selectedPosition = this.position
  },
  methods: {
    updatePosition(position) {
        this.selectedPosition = position
    },
    acceptPosition() {
        console.log("position accepted", this.position, this.selectedPosition)
        if (!this.position ||this.position.lat != this.selectedPosition.lat || this.position.lng != this.selectedPosition.lng) {
        console.log("position evented")
            this.$emit('position:updated', this.selectedPosition);
        }

    },
  }
}
</script>
