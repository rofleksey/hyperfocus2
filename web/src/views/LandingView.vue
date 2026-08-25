<script setup lang="ts">
import Button from "primevue/button";
</script>

<template>
  <section class="landing">
    <div class="hero">
      <h1>Find out when you're playing against a streamer — in Dead by Daylight</h1>
      <p class="hero-text">
        Hyperfocus watches every live <strong>Dead by Daylight</strong> stream on Twitch and reads the
        survivor names from each stream's preview. Subscribe with your Twitch and Steam profiles, and it
        will message you in your Twitch chat when your Steam name shows up in another streamer's lobby —
        often while the match is still running.
      </p>
      <div class="hero-actions">
        <RouterLink to="/subscribe"><Button label="Get notified" severity="primary" /></RouterLink>
        <RouterLink to="/live"><Button label="Browse live streams" severity="secondary" /></RouterLink>
      </div>
      <p class="hero-fineprint muted">No accounts. No cookies. Unsubscribe anytime.</p>
    </div>

    <div class="example-notification">
      <span class="example-badge">your twitch chat</span>
      <div class="chat-bubble">
        <span class="chat-author">hyperfocus</span>
        <span class="chat-message">You might be playing with @streamer</span>
      </div>
      <span class="example-hint muted">…often while the match is still going.</span>
    </div>

    <div class="section how-it-works">
      <h2>How it works</h2>
      <ol class="steps">
        <li><strong>Subscribe</strong> — enter your Twitch username and your Steam profile URL.</li>
        <li><strong>Verify</strong> — the bot joins your Twitch chat; type <code>!hyperfocussub</code> in your own channel to confirm.</li>
        <li><strong>Tracking</strong> — the four survivor names on every live DBD stream's HUD are read continuously using OCR.</li>
        <li><strong>Detected</strong> — when your Steam name matches a survivor in another streamer's lobby, you get pinged with their channel name.</li>
      </ol>
    </div>

    <div class="section limitations">
      <h2>Will it detect you?</h2>
      <p>Detection relies on reading names from stream previews, so it won't catch everything:</p>
      <ul>
        <li><strong>Anonymous mode</strong> — your nickname is replaced with the character's name, so there is nothing to match.</li>
        <li><strong>Survivor names hidden</strong> — the streamer disabled survivor nicknames in their game settings.</li>
        <li><strong>Obscured HUD</strong> — names covered by overlays, unusual HUD placement, or unreadable previews.</li>
        <li><strong>Very short matches</strong> — roughly under 5 minutes can end before the tracker checks that stream.</li>
        <li><strong>Tricky nicknames</strong> — very short, very common (like <code>cat</code>, <code>orange</code>, <code>111</code>), or non-Latin names may be missed or matched to the wrong player.</li>
      </ul>
      <p class="muted">Matching is fuzzy to tolerate OCR errors, so occasional false positives are possible — a minimum similarity threshold is applied.</p>
    </div>

    <div class="section faq">
      <h2>Good to know</h2>
      <ul>
        <li><strong>Where do notifications arrive?</strong> As chat messages in <em>your own</em> Twitch channel — the bot does not send direct messages.</li>
        <li><strong>Which games are supported?</strong> Dead by Daylight only — the name reader is built specifically for the DBD HUD.</li>
        <li><strong>How do I unsubscribe?</strong> Type <code>!hyperfocusunsub</code> in your Twitch chat. Your subscription data is deleted immediately.</li>
        <li><strong>Do I need an account?</strong> No accounts and no cookies. The only data kept is what you submit when subscribing.</li>
      </ul>
    </div>

    <div class="section history">
      <h2>Also a stream history browser</h2>
      <p>
        Hyperfocus keeps a searchable record of every live Dead by Daylight stream: browse the thumbnail
        gallery at any moment in time, search survivor names to see who appeared together, and check
        community stats.
      </p>
      <div class="hero-actions">
        <RouterLink to="/live"><Button label="Open the live gallery" severity="secondary" /></RouterLink>
        <RouterLink to="/stats"><Button label="Stats" severity="secondary" /></RouterLink>
      </div>
    </div>

    <div class="landing-footer muted">
      <RouterLink to="/about">About</RouterLink>
      <span>·</span>
      <RouterLink to="/privacy">Privacy</RouterLink>
      <span>·</span>
      <RouterLink to="/terms">Terms</RouterLink>
      <span>·</span>
      <a href="https://github.com/rofleksey/hyperfocus2" target="_blank" rel="noopener noreferrer">GitHub</a>
    </div>
  </section>
</template>

<style scoped>
.landing {
  max-width: 860px;
  margin: 0 auto;
}

.hero {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 2rem 0 1rem;
}
.hero h1 {
  margin: 0;
  font-size: 1.6rem;
  line-height: 1.25;
}
.hero-text {
  margin: 0;
  line-height: 1.6;
  color: var(--p-text-muted-color, #94a3b8);
  font-size: 0.95rem;
}
.hero-actions {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
  margin-top: 0.25rem;
}
.hero-fineprint {
  font-size: 0.75rem;
}

.example-notification {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  background: var(--p-surface-800, #1e1e2a);
  border: 1px solid var(--p-surface-700, #2d2d35);
  border-radius: 6px;
  padding: 0.9rem 1rem;
  margin: 1rem 0 2rem;
  max-width: 460px;
}
.example-badge {
  font-size: 0.65rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--p-text-muted-color, #94a3b8);
}
.chat-bubble {
  display: flex;
  gap: 0.5rem;
  align-items: baseline;
  background: rgba(255, 255, 255, 0.05);
  border-left: 3px solid var(--p-primary-color, #6366f1);
  padding: 0.5rem 0.75rem;
  border-radius: 3px;
  font-size: 0.9rem;
}
.chat-author {
  color: var(--p-primary-color, #6366f1);
  font-weight: 600;
  font-size: 0.8rem;
}
.chat-message {
  color: var(--p-text-color, #e2e8f0);
}
.example-hint {
  font-size: 0.75rem;
}

.section {
  margin-bottom: 2rem;
}
.section h2 {
  font-size: 1.05rem;
  margin: 0 0 0.5rem;
}
.section p {
  line-height: 1.6;
  font-size: 0.9rem;
  color: var(--p-text-muted-color, #94a3b8);
  margin: 0 0 0.5rem;
}
.section ul,
.section ol {
  margin: 0;
  padding-left: 1.2rem;
  line-height: 1.7;
  font-size: 0.9rem;
  color: var(--p-text-muted-color, #94a3b8);
}
.section li {
  margin-bottom: 0.35rem;
}
.section code {
  background: rgba(255, 255, 255, 0.08);
  padding: 0.1rem 0.3rem;
  border-radius: 3px;
  font-size: 0.8rem;
}
.section strong {
  color: var(--p-text-color, #e2e8f0);
}

.landing-footer {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
  font-size: 0.8rem;
  margin-top: 3rem;
  padding-top: 1rem;
  border-top: 1px solid var(--p-menubar-border-color, #2d2d35);
}
.landing-footer a {
  color: var(--p-primary-color, #6366f1);
  text-decoration: none;
}
</style>
