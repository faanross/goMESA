import { createApp } from 'vue';
import { createPinia } from 'pinia';
import Notifications from '@kyvg/vue3-notification';
import App from './App.vue';
import router from './router';

// Create Pinia store
const pinia = createPinia();

// Create Vue app
const app = createApp(App);

// Add plugins
app.use(pinia);
app.use(router);
app.use(Notifications);

// Mount app
app.mount('#app');