<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { useRoute, RouterLink } from "vue-router";
import { useHead } from "@unhead/vue";
import { ApiError, fetchSample, type Stream } from "../api";

const props = defineProps<{ streamer_id: string }>();
const route = useRoute();

const stream = ref<Stream | null>(null);
const loading = ref(false);
const notFound = ref(false);
const error = ref("");

let controller: AbortController | null = null;

const backLink = computed(() => {
  const at = route.query.at;
  return typeof at === "string" && at ? `/live?at=${encodeURIComponent(at)}` : "/live";
});

useHead({
  title: computed(() => (stream.value ? `${stream.value.display_name} — DBD stream` : "Stream")),
});

function fmt(date: string): string {
  try { return new Date(date).toLocaleString(); } catch { return date; }
}

function initials(name: string): string {
  return (name || "?").trim().charAt(0).toUpperCase();
}

async function load() {
  controller?.abort();
  const c = new AbortController();
  controller = c;
  loading.value = true;
  error.value = "";
  notFound.value = false;
  try {
    const at = (route.query.at as string) || "";
    stream.value = await fetchSample(props.streamer_id, at, c.signal);
    if (!stream.value) notFound.value = true;
  } catch (e) {
    if (c.signal.aborted) return;
    if (e instanceof ApiError && e.status === 404) {
      notFound.value = true;
    } else {
      error.value = (e as Error).message;
    }
  } finally {
    if (!c.signal.aborted) loading.value = false;
  }
}

watch(
  () => [props.streamer_id, route.query.at] as const,
  () => load(),
);
onMounted(load);
onUnmounted(() => controller?.abort());
</script>

<template>
  <section>
    <p v-if="loading" class="muted">Loading…</p>

    <div v-else-if="notFound" class="empty-state">
      <h2>Stream not found</h2>
      <p class="muted">This streamer wasn't live at that moment, or the snapshot is outside the retention window.</p>
      <RouterLink :to="backLink">&larr; Back to the gallery</RouterLink>
    </div>

    <div v-else-if="error" class="empty-state">
      <h2>Couldn't load this stream</h2>
      <p class="muted">{{ error }}</p>
      <button class="retry-btn" @click="load">Try again</button>
    </div>

    <template v-if="!loading && stream">
      <p class="muted" style="margin-bottom:0.5rem">
        <RouterLink :to="backLink">&larr; Back</RouterLink>
      </p>

      <img
        v-if="stream.preview_url"
        class="detail-thumb"
        :src="stream.preview_url"
        :alt="`${stream.display_name} Dead by Daylight stream preview`"
      />

      <div class="detail-info">
        <div class="detail-header">
          <img v-if="stream.profile_image_url" class="detail-avatar" :src="stream.profile_image_url" :alt="''" />
          <span v-else class="detail-avatar avatar-fallback" aria-hidden="true">{{ initials(stream.display_name) }}</span>
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
          <ul v-if="stream.survivor_names.length" class="survivor-list">
            <li v-for="name in stream.survivor_names" :key="name" class="survivor-chip">{{ name }}</li>
          </ul>
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
.survivor-list { list-style: none; margin: 0; padding: 0; display: flex; flex-wrap: wrap; gap: 0.4rem; }
.survivor-chip {
  padding: 0.2rem 0.6rem; background: #0b0e13; border: 1px solid #232a36; border-radius: 999px;
  font-family: ui-monospace, monospace; font-size: 0.8rem; color: #cbd5e1;
}

.empty-state { display: flex; flex-direction: column; gap: 0.5rem; align-items: flex-start; padding: 2rem 0; }
.empty-state h2 { margin: 0; font-size: 1.1rem; }
.empty-state p { margin: 0; }

.retry-btn { background: none; border: 1px solid var(--p-primary-color, #6366f1); border-radius: 3px; color: var(--p-primary-color, #6366f1); cursor: pointer; font-size: 0.8rem; padding: 0.15rem 0.4rem; }
.retry-btn:hover { background: var(--p-primary-color, #6366f1); color: #fff; }

.detail-links { display: flex; gap: 0.5rem; }
.link-btn {
  padding: 0.4rem 0.8rem; border: 1px solid var(--p-primary-color, #6366f1);
  border-radius: 4px; color: var(--p-primary-color, #6366f1); text-decoration: none; font-size: 0.85rem;
}
</style>
