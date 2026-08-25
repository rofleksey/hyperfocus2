import { createRouter, createWebHistory } from "vue-router";

declare module "vue-router" {
  interface RouteMeta {
    title?: string;
    description?: string;
    robots?: string;
  }
}

const LEGACY_QUERY_KEYS = ["at", "survivor", "q", "language", "sort", "dir"];

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/",
      name: "landing",
      component: () => import("./views/LandingView.vue"),
      meta: {
        description:
          "Hyperfocus watches live Dead by Daylight streams on Twitch and notifies you in your Twitch chat — often mid-match — when your Steam name appears in another streamer's lobby.",
      },
      beforeEnter: (to) => {
        if (LEGACY_QUERY_KEYS.some((k) => k in to.query)) {
          return { name: "live", query: { ...to.query } };
        }
      },
    },
    {
      path: "/live",
      name: "live",
      component: () => import("./views/MomentView.vue"),
      meta: {
        title: "Live DBD streams",
        description:
          "Browse every live Dead by Daylight Twitch stream: viewer counts, titles, thumbnails and OCR-read survivor names, with history navigation.",
      },
    },
    { path: "/stream/:streamer_id", name: "stream-detail", component: () => import("./views/StreamDetailView.vue"), props: true },
    {
      path: "/stats",
      name: "stats",
      component: () => import("./views/StatsView.vue"),
      meta: {
        title: "Stats",
        description: "Dead by Daylight category stats: streams online, total viewers, cycle time, disk usage, OCR success rate.",
      },
    },
    {
      path: "/subscribe",
      name: "subscribe",
      component: () => import("./views/SubscribeView.vue"),
      meta: {
        title: "Get notified",
        description:
          "Subscribe with your Twitch login and Steam profile — hyperfocus pings you in your own Twitch chat when your Steam name is spotted in another streamer's lobby.",
      },
    },
    { path: "/about", redirect: "/" },
    {
      path: "/privacy",
      name: "privacy",
      component: () => import("./views/PrivacyView.vue"),
      meta: { title: "Privacy" },
    },
    {
      path: "/terms",
      name: "terms",
      component: () => import("./views/TermsView.vue"),
      meta: { title: "Terms" },
    },
    {
      path: "/:pathMatch(.*)*",
      name: "not-found",
      component: () => import("./views/NotFoundView.vue"),
      meta: { title: "Page not found", robots: "noindex" },
    },
  ],
  scrollBehavior(to, from, savedPosition) {
    if (savedPosition) return savedPosition;
    // Query-only changes on the same route (filter tweaks) must not jump.
    if (to.path === from.path) return false;
    return { top: 0, left: 0 };
  },
});

export default router;

router.afterEach((to, from) => {
  // Move focus to the new page content on real navigations so keyboard and
  // screen-reader users land in the right place. Skipped for query-only
  // changes (filters, search) to avoid stealing focus from inputs.
  if (to.path !== from.path) {
    requestAnimationFrame(() => {
      document.getElementById("main")?.focus({ preventScroll: true });
    });
  }
});
