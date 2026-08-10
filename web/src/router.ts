import { createRouter, createWebHistory } from "vue-router";
import MomentView from "./views/MomentView.vue";
import StatsView from "./views/StatsView.vue";
import AboutView from "./views/AboutView.vue";
import PrivacyView from "./views/PrivacyView.vue";
import TermsView from "./views/TermsView.vue";

export default createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/", redirect: "/moment" },
    { path: "/moment", name: "moment", component: MomentView },
    { path: "/stats", name: "stats", component: StatsView },
    { path: "/about", name: "about", component: AboutView },
    { path: "/privacy", name: "privacy", component: PrivacyView },
    { path: "/terms", name: "terms", component: TermsView },
    { path: "/:pathMatch(.*)*", redirect: "/moment" },
  ],
});