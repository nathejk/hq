<template>
  <div class="align-middle">
    <div v-if="editing">
        <div class="form-row align-items-center">
          <div class="col">
            <input type="text" class="form-control" :class="{'form-control-sm':size=='sm'}" :placeholder="placeholder" v-model="pending" @keyup.enter="update" @keyup.esc="cancel">
          </div>
          <div class="col-auto">
            <button @click="update" type="button" class="btn btn-outline-success" :class="{'btn-sm':size=='sm'}">Opdater</button>
            <button @click="cancel" type="button" class="btn btn-outline-secondary ml-2" :class="{'btn-sm':size=='sm'}">Anuller</button>
          </div>
        </div>
    </div>
    <div v-else>
      <span>{{ value }}</span>
      <a role="button" class="btn btn-edit" :class="{'btn-sm':size=='sm'}"  @click="edit"><i class="fas fa-pencil-alt"></i></a>
    </div>
  </div>
</template>

<style lang="scss">
</style>

<script>
export default {
    props: {
        value: String,
        placeholder: String,
        type: {default:'text', type:String},
        className: String,
        size: String,
    },
    data: () => ({
        pending: '',
        editing: false,
    }),
    methods: {
      edit() {
        this.pending = this.value
        this.editing = true
      },
      cancel() {
        this.editing = false
      },
      update(event) {
        this.editing = false
        this.$emit('input', this.pending)
      },
    },
    async mounted() {
    },
}
</script>
