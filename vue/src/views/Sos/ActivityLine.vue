<template>
    <div class="media">
        <span class="avatar mr-3"></span>
        <div class="media-body line py-2">
            <span class="fa-stack fa-1x mr-2">
                <i class="fas fa-circle fa-stack-2x" :class="iconBgColorClass"></i>
                <i class="fa-stack-1x" :class="[iconClass, iconColorClass]"></i>
            </span>
            <small class="text-secondary">
                <span class="user">{{ userName }} </span>
                <slot>{{ text }}</slot>
                <span :title="timestamp"> {{ prettyTimestamp }}</span>.
            </small>
        </div>
    </div>
</template>

<style>
.media .line {
    border-left: 1px solid #ccc;
    margin-left: 20px;
}
.media .line > span {
    margin-left: -20px;
}
</style>

<script>
export default {
  props: {
    iconBgColorClass: { default:'text-lightgrey' },
    iconColorClass: { default:'text-secondary' },
    iconClass: { default:'fas fa-question' },
    userName: { default:'Nathejk' },
    text: String,
    timestamp: String,
  },
  computed: {
    prettyTimestamp() {
      const shortDays = ['Søn','Man','Tirs','Ons','Tors','Fre','Lør'];

      if (this.timestamp) {
        const ts = new Date(String(this.timestamp))
        return shortDays[ts.getDay()].toLowerCase() + ' kl. ' + ts.getHours() + ':' + ((ts.getMinutes() < 10) ? '0' : '') + ts.getMinutes()
      }
      return '-'
    },
  },
}
</script>
