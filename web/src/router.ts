import { createRouter, createWebHistory } from "vue-router";
import MomentView from "./views/MomentView.vue";
import StreamerView from "./views/StreamerView.vue";
import AboutView from "./views/AboutView.vue";
import PrivacyView from "./views/PrivacyView.vue";
import TermsView from "./views/TermsView.vue";

export default createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/", redirect: "/moment" },
    { path: "/moment", name: "moment", component: MomentView },
    { path: "/streamer/:id", name: "streamer", component: StreamerView, props: true },
    { path: "/about", name: "about", component: AboutView },
    { path: "/privacy", name: "privacy", component: PrivacyView },
    { path: "/terms", name: "terms", component: TermsView },
    { path: "/:pathMatch(.*)*", redirect: "/moment" },
  ],
});