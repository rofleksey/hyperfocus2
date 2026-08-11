<script setup lang="ts">
</script>

<template>
  <a href="#main" class="skip-link">Skip to content</a>
  <div class="app-shell">
    <header class="app-header">
      <RouterLink to="/" class="app-logo">hyperfocus</RouterLink>
      <span class="header-subtitle muted">DBD live stream history</span>
      <span style="flex:1"></span>
      <nav class="nav" aria-label="Primary">
        <RouterLink to="/subscribe"><Button icon="pi pi-bell" size="small" severity="secondary" aria-label="Notifications" title="Notifications" /></RouterLink>
        <RouterLink to="/stats"><Button icon="pi pi-chart-bar" size="small" severity="secondary" aria-label="Stats" title="Stats" /></RouterLink>
        <RouterLink to="/about"><Button icon="pi pi-info-circle" size="small" severity="secondary" aria-label="About" title="About" /></RouterLink>
      </nav>
    </header>
    <main id="main" class="app-content">
      <RouterView />
    </main>
  </div>
</template>
