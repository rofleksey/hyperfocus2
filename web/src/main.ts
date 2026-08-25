import { createApp } from "vue";
import { createHead } from "@unhead/vue/client";
import PrimeVue from "primevue/config";
import Aura from "@primevue/themes/aura";
import { definePreset } from "@primevue/themes";
import "primeicons/primeicons.css";
import "./style.css";
import App from "./App.vue";
import router from "./router";

// Aura defaults to an emerald primary, which shows up as green focus rings
// and buttons all over the dark theme. The site's design language is indigo
// (every CSS fallback uses #6366f1), so the preset is rebased on the indigo
// scale.
const HyperfocusPreset = definePreset(Aura, {
  semantic: {
    primary: {
      50: "{indigo.50}",
      100: "{indigo.100}",
      200: "{indigo.200}",
      300: "{indigo.300}",
      400: "{indigo.400}",
      500: "{indigo.500}",
      600: "{indigo.600}",
      700: "{indigo.700}",
      800: "{indigo.800}",
      900: "{indigo.900}",
      950: "{indigo.950}",
    },
  },
});

const app = createApp(App);
const head = createHead();
app.use(head);
app.use(router);
app.use(PrimeVue, {
  theme: {
    preset: HyperfocusPreset,
    options: { darkModeSelector: ".app-dark" },
  },
});
app.mount("#app");
