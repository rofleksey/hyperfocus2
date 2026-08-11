<script setup lang="ts">
import { onMounted, ref, watch } from "vue";
import { useRoute, RouterLink } from "vue-router";
import { fetchSample, type Stream } from "../api";

const props = defineProps<{ streamer_id: string }>();
const route = useRoute();

const stream = ref<Stream | null>(null);
const loading = ref(false);
const error = ref("");

function fmt(date: string): string {
  try { return new Date(date).toLocaleString(); } catch { return date; }
}

function initials(name: string): string {
  return (name || "?").trim().charAt(0).toUpperCase();
}

async function load() {
  loading.value = true;
  error.value = "";
  try {
    const at = (route.query.at as string) || "";
    stream.value = await fetchSample(props.streamer_id, at);
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}

watch(
  () => [props.streamer_id, route.query.at] as const,
  () => load(),
);
onMounted(load);
</script>

<template>
  <section>
    <p v-if="loading" class="muted">Loading…</p>
    <p v-else-if="error" class="muted">Error: {{ error }}</p>

    <template v-if="!loading && stream">
      <p class="muted" style="margin-bottom:0.5rem">
        <RouterLink to="/">&larr; Back</RouterLink>
      </p>

      <img
        v-if="stream.preview_url"
        class="detail-thumb"
        :src="stream.preview_url"
        :alt="stream.display_name"
      />

      <div class="detail-info">
        <div class="detail-header">
          <img v-if="stream.profile_image_url" class="detail-avatar" :src="stream.profile_image_url" :alt="stream.display_name" />
          <span v-else class="detail-avatar avatar-fallback">{{ initials(stream.display_name) }}</span>
          <div>
            <h2 class="detail-name">{{ stream.display_name }}</h2>
            <span class="muted">@{{ stream.login }}</span>
          </div>
        </div>

        <div class="detail-meta">
          <div class="detail-row"><span class="label">Viewers</span><span>{{ stream.viewer_count.toLocaleString() }}</span></div>
          <div class="detail-row"><span class="label">Language</span><span>{{ stream.language || '—' }}</span></div>
          <div class="detail-row"><span class="label">Started</span><span>{{ fmt(stream.started_at) }}</span></div>
          <div class="detail-row"><span class="label">Title</span><span>{{ stream.title || '—' }}</span></div>
          <div v-if="stream.tags.length" class="detail-row"><span class="label">Tags</span><span>{{ stream.tags.join(', ') }}</span></div>
        </div>

        <div class="ocr-block">
          <span class="ocr-label muted">Survivors (OCR &middot; noisy)</span>
          <pre v-if="stream.survivor_names.length" class="ocr-code"><code>{{ JSON.stringify(stream.survivor_names, null, 2) }}</code></pre>
          <span v-else class="muted">—</span>
        </div>

        <div class="detail-links">
          <a :href="`https://twitch.tv/${stream.login}`" target="_blank" rel="noopener noreferrer" class="link-btn">Watch on Twitch</a>
        </div>
      </div>
    </template>
  </section>
</template>

<style scoped>
.detail-thumb {
  width: 100%;
  aspect-ratio: 16 / 9;
  object-fit: contain;
  max-width: 1280px;
  border-radius: 6px;
  background: #000;
  display: block;
  margin-bottom: 1rem;
}

.detail-info { display: flex; flex-direction: column; gap: 1rem; }
.detail-header { display: flex; align-items: center; gap: 0.75rem; }
.detail-avatar { width: 48px; height: 48px; border-radius: 50%; }
.avatar-fallback {
  display: inline-flex; align-items: center; justify-content: center;
  width: 48px; height: 48px; background: #6366f1; color: #fff; font-size: 1.2rem; font-weight: 600; border-radius: 50%;
}
.detail-name { font-size: 1.1rem; margin: 0; }
.detail-meta { display: flex; flex-direction: column; gap: 0.5rem; }
.detail-row { display: flex; gap: 0.5rem; }
.detail-row .label { color: var(--p-text-muted-color); min-width: 80px; font-size: 0.85rem; }

.ocr-block { display: flex; flex-direction: column; gap: 0.3rem; }
.ocr-label { font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.04em; }
.ocr-code {
  margin: 0; padding: 0.6rem 0.75rem; background: #0b0e13; border: 1px solid #232a36;
  border-radius: 4px; overflow-x: auto; font-family: ui-monospace, monospace; font-size: 0.78rem;
  line-height: 1.4; color: #cbd5e1;
}

.detail-links { display: flex; gap: 0.5rem; }
.link-btn {
  padding: 0.4rem 0.8rem; border: 1px solid var(--p-primary-color, #6366f1);
  border-radius: 4px; color: var(--p-primary-color, #6366f1); text-decoration: none; font-size: 0.85rem;
}
</style>
