<script setup lang="ts">
import { ref } from 'vue';
import Button from 'primevue/button';
import InputText from 'primevue/inputtext';

const twitchLogin = ref('');
const steamURL = ref('');
const loading = ref(false);
const error = ref('');
const success = ref('');
const status = ref('');
const steamName = ref('');

async function checkStatus() {
  const login = twitchLogin.value.trim();
  if (!login) return;
  try {
    const r = await fetch(`/api/subscribe?twitch=${encodeURIComponent(login)}`);
    if (r.ok) {
      const d = await r.json();
      status.value = d.status;
      steamName.value = d.steam_name || '';
    } else {
      status.value = '';
      steamName.value = '';
    }
  } catch { status.value = ''; steamName.value = ''; }
}

async function subscribe() {
  loading.value = true; error.value = ''; success.value = '';
  try {
    const r = await fetch('/api/subscribe', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        twitch_login: twitchLogin.value.trim(),
        steam_url: steamURL.value.trim(),
      }),
    });
    const d = await r.json();
    if (r.ok) {
      success.value = d.message || 'Subscription request submitted!';
      status.value = d.status;
      steamName.value = d.steam_name || '';
    } else {
      error.value = d.error || d.message || 'Failed to subscribe';
    }
  } catch {
    error.value = 'Network error';
  } finally { loading.value = false; }
}
</script>

<template>
  <section class="subscribe-page">
    <h2>Notifications</h2>
    <p class="desc">
      Get notified in your Twitch chat when other players are detected in your DBD games.
    </p>

    <div class="form">
      <div class="field">
        <label for="tw">Twitch username</label>
        <InputText id="tw" v-model="twitchLogin" placeholder="e.g. tru3ta1ent" size="small" @blur="checkStatus" />
      </div>

      <div class="field">
        <label for="st">Steam profile URL</label>
        <InputText id="st" v-model="steamURL" placeholder="https://steamcommunity.com/id/yourname" size="small" />
      </div>

      <div v-if="status" class="status-banner" :class="'status-' + status">
        <template v-if="status === 'pending'">
          Status: <strong>Pending</strong> — type <code>!hyperfocussub</code> in your Twitch chat to verify.
        </template>
        <template v-else-if="status === 'active'">
          Status: <strong>Active</strong> — tracking your Steam name.
          <div v-if="steamName">Current name: <strong>{{ steamName }}</strong></div>
        </template>
      </div>

      <div class="actions">
        <Button label="Subscribe" severity="primary" :loading="loading" @click="subscribe" :disabled="!twitchLogin.trim() || !steamURL.trim()" />
      </div>

      <p v-if="error" class="error">{{ error }}</p>
      <p v-if="success" class="success">{{ success }}</p>
    </div>

    <div class="info">
      <h3>How it works</h3>
      <ol>
        <li>Enter your Twitch username and Steam profile URL.</li>
        <li>Submit — the bot will message you in your Twitch chat.</li>
        <li>Type <code>!hyperfocussub</code> in <strong>your own</strong> Twitch chat to verify.</li>
        <li>Done! You'll get notified when other tracked players appear in a lobby with you.</li>
      </ol>
      <p class="note">Your Steam display name is kept in sync automatically. Unverified subscriptions expire after 24 hours. Type <code>!hyperfocusunsub</code> in your chat to unsubscribe anytime.</p>
    </div>
  </section>
</template>

<style scoped>
.subscribe-page {
  max-width: 560px;
  margin: 0 auto;
}
.subscribe-page h2 { margin-top: 0; font-size: 1.1rem; }
.desc { font-size: 0.9rem; color: var(--p-text-muted-color); margin-bottom: 1.5rem; }

.form {
  display: flex; flex-direction: column; gap: 0.75rem;
  background: var(--p-surface-800, #1e1e2a);
  border: 1px solid var(--p-surface-700, #2d2d35);
  border-radius: 6px; padding: 1.25rem;
}
.field { display: flex; flex-direction: column; gap: 0.25rem; }
.field label { font-size: 0.75rem; text-transform: uppercase; letter-spacing: 0.04em; color: var(--p-text-muted-color); }
.error { color: var(--p-red-400, #f87171); font-size: 0.85rem; margin: 0; }
.success { color: var(--p-green-400, #4ade80); font-size: 0.85rem; margin: 0; }

.status-banner {
  font-size: 0.82rem; padding: 0.5rem 0.75rem; border-radius: 4px;
}
.status-pending { background: rgba(251, 191, 36, 0.12); border: 1px solid rgba(251, 191, 36, 0.3); color: #fbbf24; }
.status-active { background: rgba(34, 197, 94, 0.12); border: 1px solid rgba(34, 197, 94, 0.3); color: #4ade80; }
.status-banner code { background: rgba(255,255,255,0.1); padding: 0.1rem 0.3rem; border-radius: 3px; font-size: 0.8rem; }

.actions { display: flex; gap: 0.5rem; align-items: center; }

.info { margin-top: 2rem; font-size: 0.85rem; color: var(--p-text-muted-color); }
.info h3 { font-size: 0.95rem; margin: 0 0 0.5rem; color: var(--p-text-color); }
.info ol { padding-left: 1.2rem; line-height: 1.7; }
.info code { background: rgba(255,255,255,0.08); padding: 0.1rem 0.3rem; border-radius: 3px; font-size: 0.8rem; }
.note { margin-top: 0.75rem; font-size: 0.78rem; opacity: 0.8; }
</style>
