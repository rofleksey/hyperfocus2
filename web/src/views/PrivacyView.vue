<template>
  <section class="legal-page">
    <h2>Privacy Policy</h2>
    <p class="effective">Effective date: August 25, 2026</p>

    <h3>1. Data Collection</h3>
    <p>
      Hyperfocus does not use cookies, tracking scripts, analytics, or any
      form of user fingerprinting. No accounts are required to browse the
      site.
    </p>
    <p>
      The only personal data stored is what you submit yourself when
      subscribing to the notification bot: your Twitch login, your Steam
      profile URL, and the Steam ID and display name resolved from it.
    </p>

    <h3>2. Twitch Data</h3>
    <p>
      Hyperfocus polls the public Twitch Helix API exclusively to discover which
      broadcasters are currently live for a configured game category. The data
      retrieved — streamer display names, viewer counts, stream titles, languages,
      tags, and publicly available preview thumbnail URLs — is stored in a
      PostgreSQL database and is never shared with any third party.
    </p>

    <h3>3. Data Storage and Retention</h3>
    <p>
      Stream history (snapshots, sessions, samples) and preview thumbnail images
      are kept for a short rolling retention window on the hosted instance —
      and automatically pruned. Subscription records are kept until you
      unsubscribe via the chat command; they are deleted immediately when you
      do. You may adjust or disable retention entirely when self-hosting via
      the configuration file.
    </p>

    <h3>4. Third-Party Services</h3>
    <p>
      Two external services are contacted: the Twitch Helix API (live-stream
      discovery) and, for notification subscribers only, the Steam Web API
      (resolving the persona name behind your Steam profile). No analytics,
      crash reporting, or monitoring services are integrated.
    </p>

    <h3>5. Your Rights</h3>
    <p>
      Browsing the site leaves no user data behind. If you subscribed to
      notifications, all records tied to your subscription (Twitch login,
      Steam URL, resolved Steam name, notification history) are deleted the
      moment you run the unsubscribe chat command — or by contacting the
      operator to clear your data.
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
  margin: 0 auto;
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
