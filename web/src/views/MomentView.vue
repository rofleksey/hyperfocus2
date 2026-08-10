<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import DatePicker from "primevue/datepicker";
import InputText from "primevue/inputtext";
import Select from "primevue/select";
import Button from "primevue/button";
import Dialog from "primevue/dialog";
import { RouterLink, useRoute } from "vue-router";
import { fetchMoment, fetchSnapshots, type MomentResponse, type Snapshot, type Stream } from "../api";

const route = useRoute();
const at = ref<Date | null>(route.query.at ? new Date(route.query.at as string) : new Date());
const survivor = ref<string>("");
const q = ref<string>("");
const language = ref<string>("");
const sort = ref<string>("viewers");
const dir = ref<string>("desc");
const filtersVisible = ref(false);

const survivorSearchActive = computed(() => survivor.value.trim().length > 0);

const PAGE_SIZE = 100;

const sortOptions = [
  { label: "Viewer count", value: "viewers" },
  { label: "Name", value: "name" },
  { label: "Stream start", value: "started" },
];
const dirOptions = [
  { label: "Descending", value: "desc" },
  { label: "Ascending", value: "asc" },
];

const moment = ref<MomentResponse | null>(null);
const allStreams = ref<Stream[]>([]);
const loading = ref(false);
const loadingMore = ref(false);
const error = ref<string>("");
const hasMore = ref(true);
let offset = 0;
let debounceTimer: ReturnType<typeof setTimeout> | undefined;

const snapshots = ref<Snapshot[]>([]);

const currentSnapshotIndex = computed(() => {
  if (!moment.value?.snapshot) return -1;
  return snapshots.value.findIndex(s => s.id === moment.value!.snapshot!.id);
});
const hasPrev = computed(() => currentSnapshotIndex.value > 0);
const hasNext = computed(() => currentSnapshotIndex.value >= 0 && currentSnapshotIndex.value < snapshots.value.length - 1);

const selectedStream = ref<Stream | null>(null);
const detailVisible = ref(false);

const sentinel = ref<HTMLElement | null>(null);
let observer: IntersectionObserver | null = null;

function fmt(date: string): string {
  try {
    return new Date(date).toLocaleString();
  } catch {
    return date;
  }
}

function initials(name: string): string {
  return (name || "?").trim().charAt(0).toUpperCase();
}

function openDetail(stream: Stream) {
  selectedStream.value = stream;
  detailVisible.value = true;
}

function goPrev() {
  const idx = currentSnapshotIndex.value;
  if (idx > 0) {
    at.value = new Date(snapshots.value[idx - 1].taken_at);
  }
}

function goNext() {
  const idx = currentSnapshotIndex.value;
  if (idx >= 0 && idx < snapshots.value.length - 1) {
    at.value = new Date(snapshots.value[idx + 1].taken_at);
  }
}

async function loadFirstPage() {
  loading.value = true;
  error.value = "";
  allStreams.value = [];
  offset = 0;
  hasMore.value = true;
  try {
    const atParam = at.value ? at.value.toISOString() : "";
    moment.value = await fetchMoment(atParam, q.value.trim(), survivor.value.trim(), language.value, sort.value, dir.value, offset, PAGE_SIZE);
    allStreams.value = moment.value.streams;
    if (moment.value.streams.length < PAGE_SIZE) hasMore.value = false;
    if (snapshots.value.length === 0) {
      const snaps = await fetchSnapshots(1000);
      snapshots.value = snaps.data;
    }
  } catch (e) {
    error.value = (e as Error).message;
    moment.value = null;
  } finally {
    loading.value = false;
  }
}

async function loadMore() {
  if (loadingMore.value || !hasMore.value) return;
  loadingMore.value = true;
  offset += PAGE_SIZE;
  try {
    const atParam = at.value ? at.value.toISOString() : "";
    const page = await fetchMoment(atParam, q.value.trim(), survivor.value.trim(), language.value, sort.value, dir.value, offset, PAGE_SIZE);
    if (page.streams.length === 0 || page.streams.length < PAGE_SIZE) {
      hasMore.value = false;
    }
    allStreams.value.push(...page.streams);
  } catch (_e) {
    // silently ignore load-more errors; user can scroll-trigger retry
    offset -= PAGE_SIZE;
  } finally {
    loadingMore.value = false;
  }
}

function debounceLoad() {
  if (debounceTimer) clearTimeout(debounceTimer);
  debounceTimer = setTimeout(loadFirstPage, 300);
}

function setupObserver() {
  if (observer) observer.disconnect();
  if (!sentinel.value) return;
  observer = new IntersectionObserver((entries) => {
    if (entries[0]?.isIntersecting) {
      loadMore();
    }
  }, { rootMargin: "400px" });
  observer.observe(sentinel.value);
}

watch([survivor, q, language, sort, dir], debounceLoad);
watch(at, debounceLoad);
watch(sentinel, setupObserver);
watch(allStreams, () => {
  // Re-observe sentinel after DOM update (new items shift the sentinel).
  setTimeout(setupObserver, 0);
});

function scorePct(s: Stream): string {
  if (s.fuzzy_score == null) return "";
  return Math.round(s.fuzzy_score * 100) + "%";
}

function scoreColor(s: Stream): Record<string, string> {
  if (s.fuzzy_score == null) return {};
  const pct = s.fuzzy_score * 100;
  return { background: pct >= 50 ? "#16a34a" : "#dc2626" };
}

onMounted(() => {
  loadFirstPage();
});

onUnmounted(() => {
  if (observer) observer.disconnect();
});
</script>

<template>
  <section>
    <div class="moment-bar">
      <span class="muted moment-timestamp">
        Streamers online at
        {{ moment?.snapshot ? fmt(moment.snapshot.taken_at) : '—' }}
        <template v-if="moment?.snapshot">· {{ moment.snapshot.stream_count }} online</template>
      </span>
      <div class="survivor-search">
        <span class="pi pi-search search-icon"></span>
        <input
          class="survivor-input"
          type="text"
          v-model="survivor"
          placeholder="Search survivors…"
          autocomplete="off"
        />
        <span v-if="survivorSearchActive" class="sort-hint">relevance</span>
      </div>
      <div class="moment-nav">
        <Button icon="pi pi-chevron-left" size="small" severity="secondary" :disabled="!hasPrev" @click="goPrev" />
        <Button icon="pi pi-chevron-right" size="small" severity="secondary" :disabled="!hasNext" @click="goNext" />
      </div>
      <Button label="Filters" icon="pi pi-sliders-h" size="small" severity="secondary" @click="filtersVisible = true" />
    </div>

    <p v-if="error" class="muted">Error: {{ error }}</p>

    <div v-if="allStreams.length" class="gallery-grid">
      <div v-for="stream in allStreams" :key="stream.streamer_id" class="gallery-item">
        <div class="gallery-headline">
          <span class="gallery-name" @click="openDetail(stream)" role="button" tabindex="0">
            {{ stream.display_name }}
          </span>
          <span v-if="survivorSearchActive && stream.fuzzy_score != null" class="score-badge" :style="scoreColor(stream)" :title="'fuzzy match: ' + scorePct(stream)">
            {{ scorePct(stream) }}
          </span>
        </div>
        <a class="gallery-thumb-link" @click.prevent="openDetail(stream)" href="#">
          <img
            v-if="stream.thumb_url || stream.preview_url"
            class="gallery-thumb"
            :src="stream.thumb_url || stream.preview_url"
            :alt="stream.display_name"
            loading="lazy"
          />
          <span v-else class="muted">No preview</span>
        </a>
      </div>
    </div>

    <div v-if="loading" class="loading-spinner"><span class="spinner"></span></div>
    <p v-else-if="!loading && !allStreams.length" class="muted">No streams found for this moment.</p>

    <div v-if="allStreams.length && !loading" ref="sentinel" class="scroll-sentinel">
      <span v-if="loadingMore" class="spinner small-spinner"></span>
      <span v-else-if="!hasMore" class="muted" style="font-size:0.75rem">All streams loaded</span>
    </div>

    <Dialog v-model:visible="filtersVisible" header="Filters" :modal="true" :style="{ width: '420px', maxWidth: '95vw' }">
      <div class="filters-grid">
        <div class="field">
          <label for="at">When</label>
          <DatePicker id="at" v-model="at" showTime hourFormat="24" :showSeconds="true" dateFormat="yy-mm-dd" size="small" style="width:100%" />
        </div>
        <div class="field">
          <label for="q">Streamer name</label>
          <InputText id="q" v-model="q" placeholder="e.g. tru3" size="small" style="width:100%" />
        </div>
        <div class="field">
          <label for="language">Language</label>
          <InputText id="language" v-model="language" placeholder="e.g. en" size="small" style="width:100%" />
        </div>
        <div class="field">
          <label for="sort">Sort by</label>
          <Select id="sort" v-model="sort" :options="sortOptions" optionLabel="label" optionValue="value" size="small" style="width:100%" :disabled="survivorSearchActive" />
        </div>
        <div class="field">
          <label for="dir">Direction</label>
          <Select id="dir" v-model="dir" :options="dirOptions" optionLabel="label" optionValue="value" size="small" style="width:100%" :disabled="survivorSearchActive" />
        </div>
        <p v-if="survivorSearchActive" class="muted field-hint">Sort is disabled — survivor search ranks by relevance.</p>
      </div>
    </Dialog>

    <Dialog v-model:visible="detailVisible" :modal="true" :style="{ width: '95vw' }" @hide="selectedStream = null">
      <template #header>
        <span v-if="selectedStream">{{ selectedStream.display_name }}</span>
      </template>
      <div v-if="selectedStream" class="detail-content">
        <img
          v-if="selectedStream.preview_url"
          class="detail-thumb"
          :src="selectedStream.preview_url"
          :alt="selectedStream.display_name"
        />

        <div class="detail-info">
          <div class="detail-header">
            <img v-if="selectedStream.profile_image_url" class="detail-avatar" :src="selectedStream.profile_image_url" :alt="selectedStream.display_name" />
            <span v-else class="detail-avatar avatar-fallback">{{ initials(selectedStream.display_name) }}</span>
            <div>
              <h2 class="detail-name">
                <RouterLink :to="`/streamer/${selectedStream.streamer_id}`">{{ selectedStream.display_name }}</RouterLink>
              </h2>
              <span class="muted">@{{ selectedStream.login }}</span>
            </div>
          </div>

          <div class="detail-meta">
            <div class="detail-row">
              <span class="label">Viewers</span>
              <span>{{ selectedStream.viewer_count.toLocaleString() }}</span>
            </div>
            <div class="detail-row">
              <span class="label">Language</span>
              <span>{{ selectedStream.language || '—' }}</span>
            </div>
            <div class="detail-row">
              <span class="label">Started</span>
              <span>{{ fmt(selectedStream.started_at) }}</span>
            </div>
            <div class="detail-row">
              <span class="label">Title</span>
              <span>{{ selectedStream.title || '—' }}</span>
            </div>
            <div v-if="selectedStream.tags.length" class="detail-row">
              <span class="label">Tags</span>
              <span>{{ selectedStream.tags.join(', ') }}</span>
            </div>
          </div>

          <div class="ocr-block">
            <span class="ocr-label muted">Survivors (OCR · noisy)</span>
            <pre v-if="selectedStream.survivor_names.length" class="ocr-code"><code>{{ JSON.stringify(selectedStream.survivor_names, null, 2) }}</code></pre>
            <span v-else class="muted">—</span>
          </div>

          <div class="detail-links">
            <a v-if="selectedStream.twitch_url" :href="selectedStream.twitch_url" target="_blank" rel="noopener" class="link-btn">Watch on Twitch</a>
          </div>
        </div>
      </div>
    </Dialog>
  </section>
</template>

<style scoped>
.moment-bar {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 0.5rem;
  flex-wrap: wrap;
}

.moment-timestamp {
  font-size: 0.85rem;
  white-space: nowrap;
}

.moment-nav {
  display: flex;
  gap: 0.25rem;
}

.survivor-search {
  position: relative;
  display: flex;
  align-items: center;
  gap: 0.4rem;
  min-width: 180px;
  flex: 1;
  max-width: 320px;
}

.survivor-input {
  width: 100%;
  padding: 0.35rem 0.6rem 0.35rem 1.8rem;
  font-size: 0.85rem;
  border: 1px solid var(--p-inputtext-border-color, #3a3f4b);
  border-radius: 4px;
  background: var(--p-inputtext-background, #11151c);
  color: var(--p-inputtext-color, inherit);
}

.search-icon {
  position: absolute;
  left: 0.55rem;
  font-size: 0.8rem;
  opacity: 0.6;
  pointer-events: none;
}

.sort-hint {
  position: absolute;
  right: 0.5rem;
  font-size: 0.65rem;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  color: var(--p-primary-color, #6366f1);
  white-space: nowrap;
  pointer-events: none;
}

@media (max-width: 640px) {
  .moment-bar {
    gap: 0.5rem;
  }

  .survivor-search {
    order: 3;
    min-width: 100%;
    max-width: 100%;
  }

  .moment-nav {
    order: 2;
  }

  .moment-timestamp {
    order: 1;
    font-size: 0.75rem;
  }
}

.gallery-headline {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.4rem;
}

.score-badge {
  flex: 0 0 auto;
  font-size: 0.65rem;
  font-weight: 700;
  padding: 0.05rem 0.35rem;
  border-radius: 3px;
  color: #fff;
}

.ocr-block {
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
}

.ocr-label {
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.ocr-code {
  margin: 0;
  padding: 0.6rem 0.75rem;
  background: #0b0e13;
  border: 1px solid #232a36;
  border-radius: 4px;
  overflow-x: auto;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 0.78rem;
  line-height: 1.4;
  color: #cbd5e1;
}

.field-hint {
  font-size: 0.72rem;
  margin: 0;
}

.filters-grid {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.filters-grid .field {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.filters-grid label {
  font-size: 0.7rem;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--p-text-muted-color);
}

.gallery-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 0.75rem;
  margin-bottom: 0.5rem;
}

@media (max-width: 640px) {
  .gallery-grid {
    grid-template-columns: 1fr;
  }
}

.gallery-item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  content-visibility: auto;
  contain-intrinsic-size: auto 360px 100px;
}

.gallery-name {
  font-size: 0.8rem;
  font-weight: 600;
  color: inherit;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  cursor: pointer;
}

.gallery-thumb-link {
  display: block;
}

.gallery-thumb {
  width: 100%;
  aspect-ratio: 16 / 9;
  object-fit: cover;
  max-width: 1280px;
  border-radius: 4px;
  background: #000;
  display: block;
  cursor: pointer;
}

.detail-content {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.detail-thumb {
  width: 100%;
  aspect-ratio: 16 / 9;
  object-fit: contain;
  max-width: 1280px;
  border-radius: 6px;
  background: #000;
  display: block;
}

.detail-info {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.detail-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.detail-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
}

.avatar-fallback {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  background: #6366f1;
  color: #fff;
  font-size: 1.2rem;
  font-weight: 600;
}

.detail-name {
  font-size: 1.1rem;
  margin: 0;
}

.detail-name a {
  text-decoration: none;
  color: inherit;
}

.detail-meta {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.detail-row {
  display: flex;
  gap: 0.5rem;
}

.detail-row .label {
  color: var(--p-text-muted-color, #94a3b8);
  min-width: 80px;
  font-size: 0.85rem;
}

.detail-links {
  display: flex;
  gap: 0.5rem;
}

.link-btn {
  padding: 0.4rem 0.8rem;
  border: 1px solid var(--p-primary-color, #6366f1);
  border-radius: 4px;
  color: var(--p-primary-color, #6366f1);
  text-decoration: none;
  font-size: 0.85rem;
}

.loading-spinner {
  display: flex;
  justify-content: center;
  padding: 2rem 0;
}

.scroll-sentinel {
  display: flex;
  justify-content: center;
  padding: 1rem 0;
  min-height: 40px;
}

.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--p-surface-700, #374151);
  border-top-color: var(--p-primary-color, #6366f1);
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
}

.small-spinner {
  width: 20px;
  height: 20px;
  border-width: 2px;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
