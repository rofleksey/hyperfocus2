<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";
import { useHead } from "@unhead/vue";
import Button from "primevue/button";

const route = useRoute();

const DEFAULT_TITLE = "hyperfocus — Dead by Daylight streamer detection & alerts";
const DEFAULT_DESCRIPTION =
  "Hyperfocus watches live Dead by Daylight streams on Twitch and notifies you in your Twitch chat — often mid-match — when your Steam name appears in another streamer's lobby. Plus a browsable DBD live stream history.";

const title = computed(() => (route.meta.title ? `${route.meta.title} — hyperfocus` : DEFAULT_TITLE));

useHead({
  title,
  meta: [
    { name: "description", content: computed(() => route.meta.description || DEFAULT_DESCRIPTION) },
    { name: "robots", content: computed(() => route.meta.robots || "index") },
    { property: "og:title", content: title },
    { property: "og:description", content: computed(() => route.meta.description || DEFAULT_DESCRIPTION) },
    { property: "og:url", content: computed(() => `https://hyperfocusdbd.com${route.path}`) },
  ],
  link: [
    { rel: "canonical", href: computed(() => `https://hyperfocusdbd.com${route.path}`) },
  ],
});
</script>

<template>
  <a href="#main" class="skip-link">Skip to content</a>
  <div class="app-shell">
    <header class="app-header">
      <RouterLink to="/" class="app-logo">hyperfocus</RouterLink>
      <span class="header-subtitle muted">find yourself in Dead by Daylight streams</span>
      <span style="flex:1"></span>
      <nav class="nav" aria-label="Primary">
        <Button as="router-link" to="/live" icon="pi pi-th-large" label="Live" size="small" severity="secondary" aria-label="Live streams" title="Live streams" />
        <Button as="router-link" to="/subscribe" icon="pi pi-bell" label="Notify" size="small" severity="secondary" aria-label="Notifications" title="Notifications" />
        <Button as="router-link" to="/stats" icon="pi pi-chart-bar" label="Stats" size="small" severity="secondary" aria-label="Stats" title="Stats" />
      </nav>
    </header>
    <main id="main" class="app-content" tabindex="-1">
      <RouterView />
    </main>
  </div>
</template>

<style scoped>
.nav :deep(.p-button.router-link-active) {
  background: color-mix(in srgb, var(--p-text-color) 10%, transparent);
}

@media (max-width: 640px) {
  .nav :deep(.p-button-label) {
    display: none;
  }
}
</style>
