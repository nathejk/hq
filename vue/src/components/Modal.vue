<template>
<!-- Modal -->
<div class="modal fade" :id="id" tabindex="-1" :aria-labelledby="id + 'Title'" aria-hidden="true">
  <div class="modal-dialog modal-dialog-centered modal-dialog-scrollable modal-lg">
    <div class="modal-content">
      <div class="modal-header py-1 bg-dark text-light">
        <h5 class="modal-title font-weight-normal" :id="id + 'Title'"><i :class="iconClass"></i> {{ title }}</h5>
        <button type="button" class="close text-light" data-dismiss="modal" aria-label="Close">
          <span aria-hidden="true">&times;</span>
        </button>
      </div>
      <div class="modal-body">
        <slot />
      </div>
      <div v-if="!!$slots['modal-footer']" class="modal-footer">
        <slot name="modal-footer" />
      </div>
    </div>
  </div>
</div>
</template>

<style>
</style>

<script>

export default {
    props: {
        id: String,
        title: String,
        iconClass: String,
    },
}
</script>
