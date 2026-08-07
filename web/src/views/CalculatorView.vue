<script setup lang="ts">
import { ref } from "vue";
import InputText from "primevue/inputtext";
import Button from "primevue/button";

interface VodInfo {
  vod_id: string;
  streamer_id: string;
  started_at: string;
  duration_seconds?: number;
}

const vodInput = ref("");
const vodTimestamp = ref("");
const result = ref("");
const error = ref("");
const loading = ref(false);

function extractVodID(input: string): string | null {
  const trimmed = input.trim();
  if (!trimmed) return null;

  const urlMatch = trimmed.match(/(?:twitch\.tv\/videos\/)(\d+)/i);
  if (urlMatch) return urlMatch[1];

  if (/^\d+$/.test(trimmed)) return trimmed;

  return null;
}

function parseTimestamp(ts: string): number | null {
  const trimmed = ts.trim();
  if (!trimmed) return null;

  // Try 1h2m3s format
  const hmMatch = trimmed.match(/^(?:(\d+)h)?(?:(\d+)m)?(?:(\d+)s?)?$/i);
  if (hmMatch && (hmMatch[1] || hmMatch[2] || hmMatch[3])) {
    const h = parseInt(hmMatch[1] || '0');
    const m = parseInt(hmMatch[2] || '0');
    const s = parseInt(hmMatch[3] || '0');
    return h * 3600 + m * 60 + s;
  }

  // Try HH:MM:SS format
  const colMatch = trimmed.match(/^(\d+):(\d+):(\d+)$/);
  if (colMatch) {
    return parseInt(colMatch[1]) * 3600 + parseInt(colMatch[2]) * 60 + parseInt(colMatch[3]);
  }

  // Try raw seconds
  if (/^\d+$/.test(trimmed)) {
    return parseInt(trimmed);
  }

  return null;
}

async function calculate() {
  error.value = "";
  result.value = "";

  const vodID = extractVodID(vodInput.value);
  if (!vodID) {
    error.value = "Enter a VOD URL or numeric VOD ID";
    return;
  }

  const seconds = parseTimestamp(vodTimestamp.value);
  if (seconds === null) {
    error.value = "Enter a timestamp like 1h23m45s or HH:MM:SS";
    return;
  }

  loading.value = true;
  try {
    const res = await fetch(`/api/vods/${encodeURIComponent(vodID)}`);
    if (!res.ok) {
      if (res.status === 404) {
        error.value = "VOD not found. It may not have been captured by this tracker.";
      } else {
        error.value = `Server error: ${res.status}`;
      }
      return;
    }
    const vod: VodInfo = await res.json();

    const startedAt = new Date(vod.started_at);
    const irl = new Date(startedAt.getTime() + seconds * 1000);
    result.value = irl.toLocaleString();
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <section>
    <h2>VOD Time Calculator</h2>
    <p class="muted">Convert a VOD timestamp to real-world time.</p>

    <div class="calc-form">
      <div class="field">
        <label for="vod">VOD URL or ID</label>
        <InputText id="vod" v-model="vodInput" placeholder="https://www.twitch.tv/videos/12345 or 12345" size="small" style="width:100%" @keyup.enter="calculate" />
      </div>
      <div class="field">
        <label for="ts">Timestamp</label>
        <InputText id="ts" v-model="vodTimestamp" placeholder="e.g. 1h23m45s or 01:23:45" size="small" style="width:100%" @keyup.enter="calculate" />
      </div>
      <Button label="Calculate" icon="pi pi-calculator" size="small" :loading="loading" @click="calculate" />

      <div v-if="error" class="calc-error">{{ error }}</div>
      <div v-if="result" class="calc-result">{{ result }}</div>
    </div>
  </section>
</template>

<style scoped>
.calc-form {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  max-width: 500px;
}

.calc-form .field {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.calc-form label {
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--p-text-muted-color);
}

.calc-error {
  color: #ef4444;
  font-size: 0.85rem;
}

.calc-result {
  font-size: 1.2rem;
  font-weight: 600;
  padding: 0.75rem;
  background: var(--p-surface-800, #1e1e2a);
  border-radius: 6px;
}
</style>
