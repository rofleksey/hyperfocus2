<script setup lang="ts">
</script>

<template>
  <section class="about-page">
    <h2>About Hyperfocus</h2>

    <p>
      Hyperfocus is a self-hosted live-streamer history tracker. It periodically
      polls the Twitch API to discover which broadcasters are live for a specific
      game, records snapshots of every stream at each poll cycle — including
      viewer counts, titles, and captured preview thumbnails — and resolves VOD
      archive links so you can jump back to any moment in time.
    </p>

    <p>
      All data is stored locally in your own PostgreSQL database. Preview
      thumbnails are saved to disk. A configurable retention window automatically
      prunes old snapshots, sessions, and images. No data is ever sent to
      third-party servers beyond the initial Twitch API polling.
    </p>

    <div class="about-links">
      <RouterLink to="/privacy">Privacy Policy</RouterLink>
      <RouterLink to="/terms">Terms of Service</RouterLink>
    </div>

    <div class="about-footer">
      &copy; {{ new Date().getFullYear() }} &mdash; Made by
      <a href="https://github.com/rofleksey" target="_blank" rel="noopener">@rofleksey</a>
    </div>
  </section>
</template>

<style scoped>
.about-page {
  max-width: 640px;
}

.about-page h2 {
  margin-top: 0;
  font-size: 1.1rem;
}

.about-page p {
  line-height: 1.6;
  font-size: 0.9rem;
  color: var(--p-text-muted-color, #94a3b8);
}

.about-links {
  display: flex;
  gap: 1rem;
  margin-top: 1.5rem;
  font-size: 0.85rem;
}

.about-footer {
  margin-top: 3rem;
  padding-top: 1rem;
  border-top: 1px solid var(--p-menubar-border-color, #2d2d35);
  font-size: 0.8rem;
  color: var(--p-text-muted-color, #94a3b8);
}

.about-footer a {
  color: inherit;
}
</style>
