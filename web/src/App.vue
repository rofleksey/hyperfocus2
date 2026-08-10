<script setup lang="ts">
import { RouterLink, RouterView } from "vue-router";
import Button from "primevue/button";
</script>

<template>
  <div class="app-shell">
    <header class="app-header">
      <RouterLink to="/" class="app-logo">hyperfocus</RouterLink>
      <span class="header-subtitle muted">DBD live stream history</span>
      <span style="flex:1"></span>
      <nav class="nav">
        <RouterLink to="/about"><Button icon="pi pi-info-circle" size="small" severity="secondary" title="About" /></RouterLink>
      </nav>
    </header>
    <main class="app-content">
      <RouterView />
    </main>
  </div>
</template>