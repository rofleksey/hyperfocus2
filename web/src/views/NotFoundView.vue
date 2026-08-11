<script setup lang="ts">
import { RouterLink } from "vue-router";
</script>

<template>
  <section class="not-found">
    <h1>No such page</h1>
    <p class="sub">Skill issue.</p>
    <img src="/skill-issue.gif" alt="skill issue" class="gif" />
    <p><RouterLink to="/" class="back-link">&larr; Back to home</RouterLink></p>
  </section>
</template>

<style scoped>
.not-found { display: flex; flex-direction: column; align-items: center; justify-content: center; min-height: 60vh; text-align: center; }
.not-found h1 { font-size: 2rem; margin: 0; color: var(--p-text-color); }
.sub { color: var(--p-text-muted-color); margin: 0.25rem 0 1.5rem; font-size: 1rem; }
.gif { max-width: 360px; width: 80%; border-radius: 4px; }
.back-link { color: var(--p-primary-color); text-decoration: none; margin-top: 1rem; display: inline-block; }
</style>
