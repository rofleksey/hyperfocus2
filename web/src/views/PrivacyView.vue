<template>
  <section class="legal-page">
    <h2>Privacy Policy</h2>
    <p class="effective">Effective date: {{ new Date().getFullYear() }}</p>

    <h3>1. Data Collection</h3>
    <p>
      Hyperfocus does not collect, store, or transmit any personal information
      about its users. The application does not use cookies, tracking scripts,
      analytics, or any form of user fingerprinting.
    </p>

    <h3>2. Twitch Data</h3>
    <p>
      Hyperfocus polls the public Twitch Helix API exclusively to discover which
      broadcasters are currently live for a configured game category. The data
      retrieved — streamer display names, viewer counts, stream titles, languages,
      tags, and publicly available preview thumbnail URLs — is stored in your own
      PostgreSQL database and is never shared with any third party.
    </p>

    <h3>3. Data Storage and Retention</h3>
    <p>
      All data is stored locally on infrastructure you control. Preview thumbnail
      images are saved to disk. A configurable retention window (default: 3 days)
      automatically deletes snapshots, session records, VOD references, and
      associated image files. You may adjust or disable retention at any time via
      the configuration file.
    </p>

    <h3>4. Third-Party Services</h3>
    <p>
      The only external service contacted is the Twitch Helix API, which is
      queried on a configurable interval to obtain live-stream data. No other
      external services are contacted. No analytics, crash reporting, or
      monitoring services are integrated.
    </p>

    <h3>5. Your Rights</h3>
    <p>
      Since Hyperfocus does not collect personal data about its users, there is no
      user data to access, rectify, or delete. All stored data pertains to public
      Twitch stream information and can be deleted at any time by clearing the
      database and the local data directory.
    </p>

    <h3>6. Changes to This Policy</h3>
    <p>
      This privacy policy may be updated from time to time. Changes will be
      reflected on this page with an updated effective date.
    </p>

    <h3>7. Contact</h3>
    <p>
      For questions about this privacy policy, open an issue on the project's
      GitHub repository.
    </p>
  </section>
</template>

<style scoped>
.legal-page {
  max-width: 640px;
}

.legal-page h2 {
  margin-top: 0;
  font-size: 1.1rem;
}

.legal-page h3 {
  font-size: 0.95rem;
  margin-top: 1.5rem;
  margin-bottom: 0.25rem;
}

.legal-page p {
  line-height: 1.6;
  font-size: 0.9rem;
  color: var(--p-text-muted-color, #94a3b8);
  margin: 0.25rem 0 0.5rem;
}

.legal-page .effective {
  font-size: 0.8rem;
  margin-bottom: 1rem;
}
</style>
