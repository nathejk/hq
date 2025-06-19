import './assets/main.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import PrimeVue from 'primevue/config';
import Aura from '@primevue/themes/aura';
import ToastService from 'primevue/toastservice';



import App from './App.vue'
import router from './router'

import { library, dom } from "@fortawesome/fontawesome-svg-core";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import { fas } from '@fortawesome/free-solid-svg-icons'
import { fab } from '@fortawesome/free-brands-svg-icons';
import { far } from '@fortawesome/free-regular-svg-icons';
import { faPhone } from "@fortawesome/free-solid-svg-icons";
library.add(fas, far, fab, faPhone)
dom.watch();


localStorage.theme = 'light'

const app = createApp(App)
app.use(PrimeVue, {
    theme: {
        preset: Aura,
        options: {
            darkModeSelector: '.my-app-dark'
            //prefix: 'p',
            //darkModeSelector: 'system',
            //cssLayer: false
        }
    }
});
app.use(ToastService);
app.use(createPinia())
app.use(router)
app.component("FontAwesomeIcon", FontAwesomeIcon)

app.mount('#app')
