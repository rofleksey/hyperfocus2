import { createRouter, createWebHistory } from "vue-router";

const LEGACY_QUERY_KEYS = ["at", "survivor", "q", "language", "sort", "dir"];

export default createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/",
      name: "landing",
      component: () => import("./views/LandingView.vue"),
      beforeEnter: (to) => {
        if (LEGACY_QUERY_KEYS.some((k) => k in to.query)) {
          return { name: "live", query: { ...to.query } };
        }
      },
    },
    { path: "/live", name: "live", component: () => import("./views/MomentView.vue") },
    { path: "/stream/:streamer_id", name: "stream-detail", component: () => import("./views/StreamDetailView.vue"), props: true },
    { path: "/stats", name: "stats", component: () => import("./views/StatsView.vue") },
    { path: "/subscribe", name: "subscribe", component: () => import("./views/SubscribeView.vue") },
    { path: "/about", redirect: "/" },
    { path: "/privacy", name: "privacy", component: () => import("./views/PrivacyView.vue") },
    { path: "/terms", name: "terms", component: () => import("./views/TermsView.vue") },
    { path: "/:pathMatch(.*)*", name: "not-found", component: () => import("./views/NotFoundView.vue") },
  ],
});
