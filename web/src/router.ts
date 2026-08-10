import { createRouter, createWebHistory } from "vue-router";
import MomentView from "./views/MomentView.vue";
import StreamDetailView from "./views/StreamDetailView.vue";
import StatsView from "./views/StatsView.vue";
import AboutView from "./views/AboutView.vue";
import PrivacyView from "./views/PrivacyView.vue";
import TermsView from "./views/TermsView.vue";
import NotFoundView from "./views/NotFoundView.vue";

export default createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/", name: "moment", component: MomentView },
    { path: "/stream/:streamer_id", name: "stream-detail", component: StreamDetailView, props: true },
    { path: "/stats", name: "stats", component: StatsView },
    { path: "/about", name: "about", component: AboutView },
    { path: "/privacy", name: "privacy", component: PrivacyView },
    { path: "/terms", name: "terms", component: TermsView },
    { path: "/:pathMatch(.*)*", name: "not-found", component: NotFoundView },
  ],
});