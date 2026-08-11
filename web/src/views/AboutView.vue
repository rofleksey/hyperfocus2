<template>
  <section class="about-page">
    <h2>About Hyperfocus</h2>

    <p>
      Hyperfocus tracks <strong>Dead by Daylight</strong> live streams on Twitch.
      It takes periodic snapshots of every active streamer — capturing viewer
      counts, stream titles, preview thumbnails, and player names — so you can
      browse who was online at any moment in time.
    </p>

    <p>
      Player names are extracted from stream previews using
      <strong>optical character recognition</strong> (OCR). Type a name into the
      search bar on the main page to find which streamers that player appeared
      with — useful for tracking lobbies and snipes across the DBD community.
    </p>

    <h3>How it works</h3>

    <ul>
      <li>Twitch's API is polled continuously — each cycle fetches all live
          Dead by Daylight streams and downloads a preview thumbnail from each.</li>
      <li>Previews are processed by an OCR engine that reads the four survivor
          names from the bottom-left HUD panel.</li>
      <li>Snapshots are kept for a rolling window (currently 6 hours) and
          automatically pruned every hour.</li>
      <li>No user data is collected — there are no accounts, no cookies beyond
          what the router needs, and nothing is tracked.</li>
    </ul>

    <div class="about-links">
      <RouterLink to="/privacy">Privacy Policy</RouterLink>
      <RouterLink to="/terms">Terms of Service</RouterLink>
      <a href="https://github.com/rofleksey/hyperfocus2" target="_blank" rel="noopener noreferrer">GitHub</a>
    </div>

    <div class="about-footer">
      &copy; {{ new Date().getFullYear() }} &mdash; Made by
      <a href="https://github.com/rofleksey" target="_blank" rel="noopener noreferrer">@rofleksey</a>
    </div>
  </section>
</template>

<style scoped>
.about-page {
  max-width: 640px;
  margin: 0 auto;
}

.about-page h2 {
  margin-top: 0;
  font-size: 1.1rem;
}

.about-page h3 {
  font-size: 0.95rem;
  margin: 1.25rem 0 0.5rem;
}

.about-page p {
  line-height: 1.6;
  font-size: 0.9rem;
  color: var(--p-text-muted-color, #94a3b8);
}

.about-page ul {
  padding-left: 1.2rem;
  line-height: 1.7;
  font-size: 0.9rem;
  color: var(--p-text-muted-color, #94a3b8);
  list-style: disc;
}

.about-page ul li {
  margin-bottom: 0.25rem;
}

.about-links {
  display: flex;
  gap: 1rem;
  margin-top: 1.5rem;
  font-size: 0.85rem;
}

.about-links a {
  color: var(--p-primary-color, #6366f1);
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
