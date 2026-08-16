import './assets/main.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import PrimeVue from 'primevue/config'
import Tooltip from 'primevue/tooltip'
import Aura from '@primevue/themes/aura'
import ToastService from 'primevue/toastservice'

import App from './App.vue'
import router from './router'

import { library, dom } from '@fortawesome/fontawesome-svg-core'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { fas } from '@fortawesome/free-solid-svg-icons'
import { fab } from '@fortawesome/free-brands-svg-icons'
import { far } from '@fortawesome/free-regular-svg-icons'
import { faPhone } from '@fortawesome/free-solid-svg-icons'
import { useBuildInfo } from './composables/useBuildInfo'
import { installLiveYearSync } from './composables/useLiveYear'
library.add(fas, far, fab, faPhone)
dom.watch()

// Build info is read from <meta> tags burned into index.html at Docker build time
const buildInfo = useBuildInfo()

localStorage.theme = 'light'

const app = createApp(App)
app.use(PrimeVue, {
  locale: {
    firstDayOfWeek: 1 // 0 = Sunday, 1 = Monday
  },
  theme: {
    preset: Aura,
    options: {
      darkModeSelector: '.my-app-dark'
      //prefix: 'p',
      //darkModeSelector: 'system',
      //cssLayer: false
    }
  }
})
app.use(ToastService)
// Registered for the live-update moon in the navigation bar, which explains itself
// on hover. The native title attribute would do it, but with a delay long enough
// that an operator checking why nothing is updating gives up first.
app.directive('tooltip', Tooltip)
app.use(createPinia())
app.use(router)
app.component('FontAwesomeIcon', FontAwesomeIcon)
app.provide('buildInfo', buildInfo)

// Live updates follow the selected year: a change flushes the cache and
// resubscribes, so no data from another year can survive on screen.
installLiveYearSync()

app.mount('#app')
