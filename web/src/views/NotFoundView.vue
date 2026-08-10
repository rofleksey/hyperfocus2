<script setup lang="ts">
import { RouterLink } from "vue-router";
</script>

<template>
  <section class="not-found">
    <h1>404</h1>
    <p class="sub">The Entity claimed this page.</p>
    <div class="status-bars">
      <span class="bar bar-1"></span><span class="bar bar-2"></span><span class="bar bar-3"></span><span class="bar bar-4"></span>
    </div>
    <p><RouterLink to="/">&larr; Back to the lobby</RouterLink></p>
  </section>
</template>

<style scoped>
.not-found { display: flex; flex-direction: column; align-items: center; justify-content: center; min-height: 60vh; text-align: center; }
.not-found h1 { font-size: 5rem; margin: 0; font-weight: 800; background: linear-gradient(135deg, #6366f1, #ec4899); -webkit-background-clip: text; -webkit-text-fill-color: transparent; }
.sub { color: var(--p-text-muted-color); margin: 0.5rem 0 2rem; font-size: 0.95rem; }
.status-bars { display: flex; gap: 6px; margin-bottom: 2.5rem; }
.bar { width: 8px; border-radius: 4px; background: #374151; }
.bar-1 { height: 48px; animation: pulse 1.2s ease infinite 0s; }
.bar-2 { height: 56px; animation: pulse 1.2s ease infinite 0.15s; }
.bar-3 { height: 40px; animation: pulse 1.2s ease infinite 0.3s; }
.bar-4 { height: 52px; animation: pulse 1.2s ease infinite 0.45s; }
@keyframes pulse {
  0%, 100% { background: #374151; }
  50% { background: #6366f1; }
}
</style>
