<script setup lang="ts">
import Button from "primevue/button";
</script>

<template>
  <section class="landing">
    <div class="hero">
      <h1>Find out when you're playing against a streamer in Dead by Daylight</h1>
      <p class="hero-text">
        Hyperfocus watches every live Dead by Daylight stream on Twitch and reads the survivor
        names off each stream's preview. Subscribe with your Twitch username and Steam profile,
        and it pings you in your own Twitch chat whenever your Steam name shows up in another
        streamer's lobby, usually while the match is still going.
      </p>
      <div class="hero-actions">
        <Button as="router-link" to="/subscribe" label="Get notified" severity="primary" />
        <Button as="router-link" to="/live" label="Browse live streams" severity="secondary" />
      </div>
      <p class="hero-fineprint muted">No accounts. No cookies. Unsubscribe anytime.</p>
    </div>

    <div class="example-notification">
      <span class="example-badge">your twitch chat</span>
      <div class="chat-bubble">
        <span class="chat-author">hyperfocus</span>
        <span class="chat-message">You might be playing with @streamer</span>
      </div>
      <span class="example-hint muted">That's it. No spam, just a ping when it happens.</span>
    </div>

    <div class="section how-it-works">
      <h2>How it works</h2>
      <p>
        Hyperfocus asks Twitch for all live Dead by Daylight streams around the clock. For each
        one it downloads a 1080p preview image (720p when that's all that exists) and reads the
        four survivor names from the HUD panel using OCR. If one of those names matches yours,
        the bot pings you in your own Twitch chat.
      </p>
      <p>Setting it up takes a minute:</p>
      <ol class="steps">
        <li>Enter your Twitch username and your Steam profile URL.</li>
        <li>The bot joins your Twitch chat. Type <code>!hyperfocussub</code> in your own channel to confirm it's really you.</li>
        <li>Done. When your Steam name is spotted in another streamer's lobby, you get pinged with their channel name.</li>
      </ol>
    </div>

    <div class="section limitations">
      <h2>Will it detect you?</h2>
      <p>It reads names off stream previews, so sometimes it simply can't see you:</p>
      <ul>
        <li><strong>Anonymous mode</strong> replaces your nickname with the character's name, so there is nothing to match.</li>
        <li><strong>Hidden survivor names</strong>: if the streamer turned them off in their game settings, the names never appear on screen.</li>
        <li><strong>Blocked HUD</strong>: overlays, unusual HUD placement, or a low quality preview can cover the names.</li>
        <li><strong>Short matches</strong>: a match under roughly 5 minutes can end before the tracker looks at that stream.</li>
        <li><strong>Tricky nicknames</strong>: very short or very common names (like <code>cat</code>, <code>orange</code>, <code>111</code>) and non-Latin names may be missed or matched to the wrong player.</li>
      </ul>
      <p class="muted">Matching is fuzzy on purpose so OCR mistakes don't cause misses, which means a false positive is possible every now and then.</p>
    </div>

    <div class="section faq">
      <h2>Good to know</h2>
      <ul>
        <li><strong>Where do notifications arrive?</strong> In your own Twitch chat, as a message from the bot. No direct messages.</li>
        <li><strong>Which games are supported?</strong> Dead by Daylight only. The name reader is built for the DBD HUD.</li>
        <li><strong>How do I unsubscribe?</strong> Type <code>!hyperfocusunsub</code> in your Twitch chat. Your subscription data is deleted right away.</li>
        <li><strong>Do I need an account?</strong> No accounts and no cookies. The only data kept is what you submit when subscribing.</li>
      </ul>
    </div>

    <div class="section history">
      <h2>A searchable archive of every stream</h2>
      <p>
        Hyperfocus also keeps a record of every live Dead by Daylight stream it has seen. Browse
        the thumbnail gallery at any moment in time, search a survivor name to see who played
        together, or check community stats. Snapshots are kept for a rolling window (3 hours
        on hyperfocusdbd.com, configurable when self-hosting) and pruned automatically.
      </p>
    </div>

    <div class="landing-footer">
      <p class="legal muted">
        Hyperfocus is an independent fan project. It is not affiliated with, endorsed by, or
        sponsored by Twitch Interactive, Valve, or Behaviour Interactive. Dead by Daylight,
        Steam, Twitch, and all related trademarks belong to their respective owners.
      </p>
      <div class="footer-links muted">
        <RouterLink to="/privacy">Privacy</RouterLink>
        <span>·</span>
        <RouterLink to="/terms">Terms</RouterLink>
        <span>·</span>
        <a href="https://github.com/rofleksey/hyperfocus2" target="_blank" rel="noopener noreferrer">GitHub</a>
        <span>·</span>
        <span>&copy; {{ new Date().getFullYear() }} made by <a href="https://github.com/rofleksey" target="_blank" rel="noopener noreferrer">@rofleksey</a></span>
      </div>
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
  flex-direction: column;
  gap: 0.75rem;
  font-size: 0.8rem;
  margin-top: 3rem;
  padding-top: 1rem;
  border-top: 1px solid var(--p-menubar-border-color, #2d2d35);
}
.legal {
  margin: 0;
  font-size: 0.72rem;
  line-height: 1.5;
}
.footer-links {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
  align-items: baseline;
}
.footer-links a {
  color: var(--p-primary-color, #6366f1);
  text-decoration: none;
}
</style>
