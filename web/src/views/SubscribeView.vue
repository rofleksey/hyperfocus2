<script setup lang="ts">
import { computed, onUnmounted, ref } from 'vue';
import Button from 'primevue/button';
import InputText from 'primevue/inputtext';
import { ApiError, fetchSubscriptionStatus, postSubscription } from '../api';

const twitchLogin = ref('');
const steamURL = ref('');
const loading = ref(false);
const error = ref('');
const success = ref('');
const status = ref('');
const steamName = ref('');

let statusController: AbortController | null = null;

const steamUrlValid = computed(() => {
  const v = steamURL.value.trim();
  if (!v) return true; // required-ness is handled by the submit guard
  return /^https?:\/\/steamcommunity\.com\/(id\/[A-Za-z0-9_-]{2,32}|profiles\/\d{17})(\/[^\s]*)?$/i.test(v);
});

const canSubmit = computed(
  () => twitchLogin.value.trim().length > 0 && steamURL.value.trim().length > 0 && steamUrlValid.value && !loading.value,
);

async function checkStatus() {
  const login = twitchLogin.value.trim();
  if (!login) return;
  statusController?.abort();
  statusController = new AbortController();
  try {
    const d = await fetchSubscriptionStatus(login, statusController.signal);
    status.value = d.status;
  } catch {
    status.value = '';
  }
}

async function subscribe() {
  if (!canSubmit.value) return;
  loading.value = true; error.value = ''; success.value = '';
  try {
    const d = await postSubscription(twitchLogin.value.trim(), steamURL.value.trim());
    success.value = d.message || 'Subscription request submitted!';
    status.value = d.status;
    steamName.value = d.steam_name || '';
  } catch (e) {
    error.value = e instanceof ApiError ? e.message : 'Network error — please try again.';
  } finally { loading.value = false; }
}

onUnmounted(() => statusController?.abort());
</script>

<template>
  <section class="subscribe-page">
    <h2>Notifications</h2>
    <p class="desc">
      Get a heads-up in your Twitch chat when your Steam name is spotted in another streamer's
      Dead by Daylight lobby — often mid-match.
    </p>

    <form class="form" @submit.prevent="subscribe">
      <div class="field">
        <label for="tw">Twitch username</label>
        <InputText id="tw" v-model="twitchLogin" placeholder="e.g. tru3ta1ent" size="small" autocomplete="off" @blur="checkStatus" />
      </div>

      <div class="field">
        <label for="st">Steam profile URL</label>
        <InputText id="st" v-model="steamURL" placeholder="https://steamcommunity.com/id/yourname" size="small" autocomplete="off" inputmode="url" :aria-invalid="!steamUrlValid ? 'true' : undefined" />
        <small v-if="!steamUrlValid" class="field-error">Enter a full Steam profile URL, e.g. https://steamcommunity.com/id/yourname</small>
      </div>

      <div v-if="status" class="status-banner" :class="'status-' + status" role="status">
        <template v-if="status === 'pending'">
          Status: <strong>Pending</strong> — type <code>!hyperfocussub</code> in your Twitch chat to verify.
        </template>
        <template v-else-if="status === 'active'">
          Status: <strong>Active</strong> — tracking your Steam name.
          <div v-if="steamName">Current name: <strong>{{ steamName }}</strong></div>
        </template>
      </div>

      <div class="actions">
        <Button type="submit" label="Subscribe" severity="primary" :loading="loading" :disabled="!canSubmit" />
      </div>

      <p v-if="error" class="error" role="alert">{{ error }}</p>
      <p v-if="success" class="success">{{ success }}</p>
    </form>

    <div class="info">
      <h3>How it works</h3>
      <ol>
        <li>Enter your Twitch username and Steam profile URL.</li>
        <li>Submit — the bot will message you in your Twitch chat.</li>
        <li>Type <code>!hyperfocussub</code> in <strong>your own</strong> Twitch chat to verify.</li>
        <li>Done! When your Steam name is spotted in another streamer's lobby, the bot pings you with their Twitch username, e.g. <code>You might be playing with @streamer</code>.</li>
      </ol>
      <p class="note">Your Steam display name is kept in sync automatically. Unverified subscriptions expire after 24 hours. Type <code>!hyperfocusunsub</code> in your chat to unsubscribe anytime.</p>

      <h3>Where do notifications arrive?</h3>
      <p>As chat messages in your <strong>own</strong> Twitch channel. The bot does not send direct messages.</p>

      <h3>Will it detect you?</h3>
      <ul>
        <li>Anonymous mode replaces your nickname with the character name — undetectable.</li>
        <li>Streamers can hide survivor nicknames in their game settings.</li>
        <li>Names covered by overlays or unusual HUD layouts may be unreadable.</li>
        <li>Matches under ~5 minutes can end before a detection cycle.</li>
        <li>Very short, common, or non-Latin nicknames may be missed or matched to the wrong player.</li>
      </ul>
      <p class="note">Dead by Daylight only.</p>
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
.field-error { color: var(--p-red-400, #f87171); font-size: 0.72rem; }
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
.info ol, .info ul { padding-left: 1.2rem; line-height: 1.7; }
.info code { background: rgba(255,255,255,0.08); padding: 0.1rem 0.3rem; border-radius: 3px; font-size: 0.8rem; }
.note { margin-top: 0.75rem; font-size: 0.78rem; opacity: 0.8; }
</style>
