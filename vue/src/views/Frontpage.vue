<template>
    <div class="container">
            <div class="row">
                <div class="col rounded m-2 py-2 hazyblue" style="background:#a2aeb2; ">
                    <i class="fas fa-4x fa-flag"></i>
                    Aktive
                </div>
                <div class="col bg-secondary text-light rounded m-2">
                    Ankommet
                </div>
                <div class="col bg-light ext-light rounded m-2">
                    Mangler
                </div>
                <div class="col bg-light ext-light rounded m-2">
                    Udgåede
                </div>
            </div>
        Forside
    </div>
</template>

<style>
</style>

<script>
//import '@/assets/main.scss'
import moment from 'moment'

export default {
    data: () => ({
      title: 'Nathejk 2019',
      countdownDate: moment('2019-04-26 22:32'),
      now: moment(),
      tick: null,
    }),
    computed: {
        countdownActive: function() {
            //return this.countdownDate.isBefore(moment());
            return moment().isBefore(this.countdownDate);
        }
    },
    methods: {
      afterEnter: function () {
        // Scroll to top on page switch
        window.scrollTo({top: 0, behavior: 'smooth'})
      },
    },
    mounted: function () {
        this.tick = setInterval(function () {
            this.now = moment();
        }.bind(this), 1000)
    },
    beforeDestroy() {
        clearInterval(this.tick);
    }
}
</script>
