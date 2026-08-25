<script setup lang="ts">
import Button from "primevue/button";
</script>

<template>
  <a href="#main" class="skip-link">Skip to content</a>
  <div class="app-shell">
    <header class="app-header">
      <RouterLink to="/" class="app-logo">hyperfocus</RouterLink>
      <span class="header-subtitle muted">find yourself in Dead by Daylight streams</span>
      <span style="flex:1"></span>
      <nav class="nav" aria-label="Primary">
        <RouterLink to="/live"><Button icon="pi pi-th-large" label="Live" size="small" severity="secondary" aria-label="Live streams" title="Live streams" /></RouterLink>
        <RouterLink to="/subscribe"><Button icon="pi pi-bell" label="Notify" size="small" severity="secondary" aria-label="Notifications" title="Notifications" /></RouterLink>
        <RouterLink to="/stats"><Button icon="pi pi-chart-bar" label="Stats" size="small" severity="secondary" aria-label="Stats" title="Stats" /></RouterLink>
        <RouterLink to="/about"><Button icon="pi pi-info-circle" label="About" size="small" severity="secondary" aria-label="About" title="About" /></RouterLink>
      </nav>
    </header>
    <main id="main" class="app-content">
      <RouterView />
    </main>
  </div>
</template>

<style scoped>
@media (max-width: 640px) {
  .nav :deep(.p-button-label) {
    display: none;
  }
}
</style>
