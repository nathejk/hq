<template>
  <div class="row">
    <div v-for="department in departments" class="col-3">
        <h3>{{ department.name }}</h3>
      <draggable class="list-group" ghost-class="ghost" :list="department.units" group="people" @change="log">
        <div
          class="list-group-item"
          v-for="(unit, index) in department.units"
          :key="unit.name"
        >
            <div v-if="unit.edit">
                <form>
                  <div class="form-row align-items-center">
                    <div class="col">
                      <input type="text" class="form-control" placeholder="Hold" v-model="unit.edit">
                    </div>
                    <div class="col-auto">
                      <button click="saveHeadline" type="button" class="btn btn-sm btn-hint"><i class="fas fa-check"></i></button>
                      <button @click="unit.edit=false" type="button" class="btn btn-sm btn-hint ml-2"><i class="fas fa-times"></i></button>
                    </div>
                  </div>
                </form>
            </div>
            <div v-else class="d-flex justify-content-between">
                {{ unit.name }} {{ index }}
                <button type="button" class="btn btn-sm btn-hint" @click="unit.edit=true"><i class="fas fa-pencil-alt"></i></button>
                <span v-if="false" class="text-muted">#12</span>
            </div>
        </div>
      </draggable>
      <button class="btn btn-outline-secondary" @click="department.units.push({edit:false})">+</button>
    </div>

  </div>
</template>

<style scoped>
.buttons {
  margin-top: 35px;
}
.ghost {
  opacity: 0.5;
  background: #c8ebfb;
}
</style>

<script>
import draggable from "vuedraggable";
export default {
  name: "two-lists",
  display: "Two Lists",
  order: 1,
  components: {
    draggable
  },
  data: () => ({
      departments: [
        {name:'Banditter', slug:'bandit', units:[{id:1, name:"One", edit:false}]},
        {name:'Postmandskab', slug:'post', units:[]},
        {name:'Logistik', slug:'logistik', units:[]},
        {name:'Guides', slug:'guide', units:[]},
        {name:'Andet', slug:'', units:[]},
      ],
  }),

  methods: {
    add: function() {
      this.list.push({ name: "Juan" });
    },
    replace: function() {
      this.list = [{ name: "Edgard" }];
    },
    clone: function(el) {
      return {
        name: el.name + " cloned"
      };
    },
    log: function(evt) {
      window.console.log(evt);
    }
  }
};
</script>
