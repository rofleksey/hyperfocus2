import { createApp } from "vue";
import { createHead } from "@unhead/vue/client";
import PrimeVue from "primevue/config";
import Aura from "@primevue/themes/aura";
import "primeicons/primeicons.css";
import "./style.css";
import App from "./App.vue";
import router from "./router";

const app = createApp(App);
const head = createHead();
app.use(head);
app.use(router);
app.use(PrimeVue, {
  theme: {
    preset: Aura,
    options: { darkModeSelector: ".app-dark" },
  },
});
app.mount("#app");
