import Vue from "vue"

Vue.filter("first4Chars", str => str.substring(0, 4))
Vue.filter("last4Chars", str => str.substring(str.length - 4))

Vue.filter('capitalize', function (value) {
  if (!value) return ''
  value = value.toString()
  return value.charAt(0).toUpperCase() + value.slice(1)
})

const korps = {
    'dds': 'Det Danske Spejderkorps',
    'kfum': 'KFUM-Spejderne',
    'kfuk': 'De grønne pigespejdere',
    'dbs': 'Danske Baptisters Spejderkorps',
    'dgs': 'De Gule Spejdere',
    'dss': 'Dansk Spejderkorps Sydslesvig',
    'fdf': 'FDF / FPF',
    'andet': 'Andet',
}
Vue.filter("korps", slug => korps[slug])

import moment from 'moment';

Vue.filter('formatDate', function(value) {
    moment.locale('da')
    if (value) {
        return moment(String(value)).format('ddd [d.] DD. HH:mm')
    }
});
