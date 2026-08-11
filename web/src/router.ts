import { createRouter, createWebHistory } from "vue-router";

export default createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/", name: "moment", component: () => import("./views/MomentView.vue") },
    { path: "/stream/:streamer_id", name: "stream-detail", component: () => import("./views/StreamDetailView.vue"), props: true },
    { path: "/stats", name: "stats", component: () => import("./views/StatsView.vue") },
    { path: "/subscribe", name: "subscribe", component: () => import("./views/SubscribeView.vue") },
    { path: "/about", name: "about", component: () => import("./views/AboutView.vue") },
    { path: "/privacy", name: "privacy", component: () => import("./views/PrivacyView.vue") },
    { path: "/terms", name: "terms", component: () => import("./views/TermsView.vue") },
    { path: "/:pathMatch(.*)*", name: "not-found", component: () => import("./views/NotFoundView.vue") },
  ],
});
