<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from "vue";
import DatePicker from "primevue/datepicker";
import InputText from "primevue/inputtext";
import Select from "primevue/select";
import Button from "primevue/button";
import Dialog from "primevue/dialog";
import { useRoute, useRouter } from "vue-router";
import { fetchMoment, fetchSnapshots, type MomentResponse, type Snapshot, type Stream } from "../api";

const SORT_OPTIONS = ["viewers", "name", "started"] as const;
const DIR_OPTIONS = ["asc", "desc"] as const;
const PAGE_SIZE = 100;
const RETENTION_HOURS = 6;
const DEBOUNCE_MS = 300;
const POLL_INTERVAL_MS = 30000;
const SNAPSHOTS_LIMIT = 200;

function qStr(v: unknown): string {
  if (Array.isArray(v)) return String(v[0] ?? "");
  return String(v ?? "");
}

const route = useRoute();
const router = useRouter();
const at = ref<Date | null>(route.query.at ? new Date(qStr(route.query.at)) : new Date());
const survivor = ref<string>(qStr(route.query.survivor));
const q = ref<string>(qStr(route.query.q));
const language = ref<string>(qStr(route.query.language));
const sort = ref<string>(SORT_OPTIONS.includes(qStr(route.query.sort) as typeof SORT_OPTIONS[number]) ? qStr(route.query.sort) : "viewers");
const dir = ref<string>(DIR_OPTIONS.includes(qStr(route.query.dir) as typeof DIR_OPTIONS[number]) ? qStr(route.query.dir) : "desc");
const filtersVisible = ref(false);

const survivorSearchActive = computed(() => survivor.value.trim().length > 0);

const hasFilters = computed(() => q.value.trim() !== "" || language.value.trim() !== "" || sort.value !== "viewers" || dir.value !== "desc");

const sortOptions = [
  { label: "Viewer count", value: "viewers" },
  { label: "Name", value: "name" },
  { label: "Stream start", value: "started" },
];
const dirOptions = [
  { label: "Descending", value: "desc" },
  { label: "Ascending", value: "asc" },
];

const outsideRetention = computed(() => {
  if (!at.value) return false;
  const cutoff = new Date(Date.now() - RETENTION_HOURS * 3600 * 1000);
  return at.value < cutoff;
});

const moment = ref<MomentResponse | null>(null);
const allStreams = ref<Stream[]>([]);
const loading = ref(false);
const loadingMore = ref(false);
const error = ref<string>("");
const hasMore = ref(true);
let offset = 0;
let debounceTimer: ReturnType<typeof setTimeout> | undefined;
let syncingFromRoute = 0;
let momentController: AbortController | null = null;

const snapshots = ref<Snapshot[]>([]);
const latestSnapshotAt = ref<string>("");
const lastSeenLatestAt = ref<string>("");
const hasNewerSnapshot = computed(() => {
  if (!lastSeenLatestAt.value || !latestSnapshotAt.value) return false;
  return latestSnapshotAt.value > lastSeenLatestAt.value;
});

let snapshotPollTimer: ReturnType<typeof setInterval> | undefined;

const currentSnapshotIndex = computed(() => {
  if (!moment.value?.snapshot) return -1;
  return snapshots.value.findIndex(s => s.id === moment.value!.snapshot!.id);
});
const hasPrev = computed(() => currentSnapshotIndex.value >= 0 && currentSnapshotIndex.value < snapshots.value.length - 1);
const hasNext = computed(() => currentSnapshotIndex.value > 0);

const sentinel = ref<HTMLElement | null>(null);
let observer: IntersectionObserver | null = null;

function fmt(date: string): string {
  try { return new Date(date).toLocaleString(); } catch { return date; }
}

function goNow() {
  lastSeenLatestAt.value = latestSnapshotAt.value;
  at.value = new Date();
}

function goPrev() {
  const idx = currentSnapshotIndex.value;
  if (idx >= 0 && idx < snapshots.value.length - 1) { at.value = new Date(snapshots.value[idx + 1].taken_at); }
}

function goNext() {
  const idx = currentSnapshotIndex.value;
  if (idx > 0) { at.value = new Date(snapshots.value[idx - 1].taken_at); }
}

async function loadFirstPage() {
  momentController?.abort();
  momentController = new AbortController();
  const signal = momentController.signal;
  loading.value = true; error.value = ""; offset = 0; hasMore.value = true;
  allStreams.value = [];
  try {
    const atParam = at.value ? at.value.toISOString() : "";
    moment.value = await fetchMoment(atParam, q.value.trim(), survivor.value.trim(), language.value, sort.value, dir.value, offset, PAGE_SIZE, signal);
    allStreams.value = moment.value.streams;
    if (moment.value.streams.length < PAGE_SIZE) hasMore.value = false;
    const snaps = await fetchSnapshots(SNAPSHOTS_LIMIT, signal);
    snapshots.value = snaps.data;
    await checkLatest(signal);
    window.scrollTo({ top: 0, behavior: "smooth" });
  } catch (e) {
    if ((e as Error).name !== "AbortError") {
      error.value = (e as Error).message;
      moment.value = null;
    }
  } finally { loading.value = false; }
}

async function loadMore() {
  if (loadingMore.value || !hasMore.value) return;
  loadingMore.value = true; offset += PAGE_SIZE;
  try {
    const atParam = at.value ? at.value.toISOString() : "";
    const page = await fetchMoment(atParam, q.value.trim(), survivor.value.trim(), language.value, sort.value, dir.value, offset, PAGE_SIZE, momentController?.signal);
    if (page.streams.length < PAGE_SIZE) hasMore.value = false;
    allStreams.value.push(...page.streams);
  } catch (e) {
    if ((e as Error).name !== "AbortError") {
      offset -= PAGE_SIZE;
      hasMore.value = false;
    }
  } finally { loadingMore.value = false; }
}

async function checkLatest(signal?: AbortSignal) {
  try {
    const snaps = await fetchSnapshots(1, signal);
    if (snaps.data.length) {
      const latest = snaps.data[0].taken_at;
      if (!lastSeenLatestAt.value) lastSeenLatestAt.value = latest;
      latestSnapshotAt.value = latest;
    }
  } catch (_e) {}
}

function debounceLoad() {
  if (syncingFromRoute > 0) return;
  if (debounceTimer) clearTimeout(debounceTimer);
  debounceTimer = setTimeout(loadFirstPage, DEBOUNCE_MS);
}

function setupObserver() {
  if (observer) observer.disconnect();
  if (!sentinel.value) return;
  observer = new IntersectionObserver((entries) => {
    if (entries.some(e => e.isIntersecting) && !loadingMore.value) loadMore();
  }, { rootMargin: "400px" });
  observer.observe(sentinel.value);
}

function syncURL() {
  if (syncingFromRoute > 0) return;
  const params: Record<string, string | undefined> = {};
  if (at.value) params.at = at.value.toISOString();
  if (survivor.value.trim()) params.survivor = survivor.value.trim();
  if (q.value.trim()) params.q = q.value.trim();
  if (language.value.trim()) params.language = language.value.trim();
  if (sort.value !== "viewers") params.sort = sort.value;
  if (dir.value !== "desc") params.dir = dir.value;
  router.replace({ query: { ...params } });
}

watch(() => route.fullPath, async () => {
  syncingFromRoute++;
  if (Object.keys(route.query).length === 0) {
    at.value = new Date();
    survivor.value = "";
    q.value = "";
    language.value = "";
    sort.value = "viewers";
    dir.value = "desc";
  } else {
    if (route.query.at) at.value = new Date(qStr(route.query.at));
    survivor.value = qStr(route.query.survivor);
    q.value = qStr(route.query.q);
    language.value = qStr(route.query.language);
    sort.value = SORT_OPTIONS.includes(qStr(route.query.sort) as typeof SORT_OPTIONS[number]) ? qStr(route.query.sort) : "viewers";
    dir.value = DIR_OPTIONS.includes(qStr(route.query.dir) as typeof DIR_OPTIONS[number]) ? qStr(route.query.dir) : "desc";
  }
  await nextTick();
  syncingFromRoute--;
});

watch(at, syncURL);
watch(survivor, syncURL);
watch(q, syncURL);
watch(language, syncURL);
watch(sort, syncURL);
watch(dir, syncURL);
watch(at, debounceLoad);
watch([survivor, q, language, sort, dir], debounceLoad);
watch(sentinel, setupObserver);

function scorePct(s: Stream): string {
  if (s.fuzzy_score == null) return "";
  return Math.round(s.fuzzy_score * 100) + "%";
}

function scoreColor(s: Stream): Record<string, string> {
  if (s.fuzzy_score == null) return {};
  return { background: s.fuzzy_score * 100 >= 50 ? "#16a34a" : "#dc2626" };
}

function streamLink(stream: Stream): string {
  const p = new URLSearchParams();
  if (at.value) p.set("at", at.value.toISOString());
  return `/stream/${stream.streamer_id}?${p.toString()}`;
}

function isDocumentHidden() {
  return document.visibilityState === "hidden";
}

onMounted(() => {
  if (!route.query.at) syncURL();
  loadFirstPage();
  function safePoll() { if (!isDocumentHidden()) checkLatest(); }
  snapshotPollTimer = setInterval(safePoll, POLL_INTERVAL_MS);
});

onUnmounted(() => {
  momentController?.abort();
  if (observer) observer.disconnect();
  if (snapshotPollTimer) clearInterval(snapshotPollTimer);
  if (debounceTimer) clearTimeout(debounceTimer);
});
</script>

<template>
  <section>
    <div class="moment-bar">
      <span class="muted moment-timestamp">
        Online at {{ moment?.snapshot ? fmt(moment.snapshot.taken_at) : '—' }}
        <template v-if="moment?.snapshot">· {{ moment.snapshot.stream_count }} online</template>
      </span>
      <span class="moment-spacer"></span>
      <div class="moment-controls">
        <div class="survivor-search">
          <span class="pi pi-search search-icon" aria-hidden="true"></span>
          <label for="survivor-search" class="sr-only">Search survivors</label>
          <input id="survivor-search" class="survivor-input" type="text" v-model="survivor" placeholder="Search survivors…" autocomplete="off" />
          <button v-if="survivor" class="input-clear pi pi-times" @click="survivor = ''" aria-label="Clear survivor search"></button>
          <span v-if="survivorSearchActive" class="sort-hint">relevance</span>
        </div>
        <div class="moment-nav">
          <Button icon="pi pi-chevron-left" size="small" severity="secondary" :disabled="!hasPrev" @click="goPrev" aria-label="Previous snapshot" />
          <Button icon="pi pi-chevron-right" size="small" severity="secondary" :disabled="!hasNext" @click="goNext" aria-label="Next snapshot" />
          <Button icon="pi pi-arrow-right" label="Now" size="small" severity="secondary" :class="{ 'now-btn': true, 'now-glow': hasNewerSnapshot }" @click="goNow" :aria-label="hasNewerSnapshot ? 'Jump to now (newer snapshot available)' : 'Jump to now'" />
          <Button icon="pi pi-sliders-h" label="Filters" size="small" :severity="hasFilters ? 'primary' : 'secondary'" class="filter-btn" @click="filtersVisible = true" aria-label="Open filters" />
        </div>
      </div>
    </div>

    <p v-if="outsideRetention" class="retention-warn">
      This time is outside the {{ RETENTION_HOURS }}-hour retention window. Data may be incomplete or missing.
    </p>

    <p v-if="error" class="muted">Error: {{ error }} <button class="retry-btn" @click="loadFirstPage">Try again</button></p>

    <div v-if="allStreams.length" class="gallery-grid" role="list">
      <div v-for="stream in allStreams" :key="stream.streamer_id" class="gallery-item" role="listitem">
        <div class="gallery-headline">
          <RouterLink class="gallery-name" :to="streamLink(stream)" :title="stream.display_name">{{ stream.display_name }}</RouterLink>
          <span v-if="survivorSearchActive && stream.fuzzy_score != null" class="score-badge" :style="scoreColor(stream)" :title="'match: ' + scorePct(stream)">{{ scorePct(stream) }}</span>
        </div>
        <RouterLink class="gallery-thumb-link" :to="streamLink(stream)">
          <img v-if="stream.thumb_url || stream.preview_url" class="gallery-thumb" :src="stream.thumb_url || stream.preview_url" :alt="stream.display_name" loading="lazy" decoding="async" />
          <span v-else class="muted">No preview</span>
        </RouterLink>
      </div>
    </div>

    <div v-if="loading" class="loading-spinner"><span class="spinner"></span></div>
    <p v-else-if="!loading && !allStreams.length" class="muted">
      No streams found for this moment. <button class="retry-btn" @click="loadFirstPage">Try again</button> or <button class="retry-btn" @click="goNow">jump to now</button>.
    </p>

    <div v-if="allStreams.length && !loading" ref="sentinel" class="scroll-sentinel">
      <span v-if="loadingMore" class="spinner small-spinner"></span>
      <button v-else-if="!hasMore && allStreams.length > PAGE_SIZE" class="retry-btn" @click="loadFirstPage" style="font-size:0.75rem">All streams loaded — scroll to top</button>
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
          <div class="input-wrap">
            <InputText id="q" v-model="q" placeholder="e.g. tru3" size="small" style="width:100%" />
            <button v-if="q" class="input-clear pi pi-times" @click="q = ''" aria-label="Clear streamer name"></button>
          </div>
        </div>
        <div class="field">
          <label for="language">Language</label>
          <div class="input-wrap">
            <InputText id="language" v-model="language" placeholder="e.g. en" size="small" style="width:100%" />
            <button v-if="language" class="input-clear pi pi-times" @click="language = ''" aria-label="Clear language"></button>
          </div>
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
        <Button v-if="hasFilters" label="Reset filters" severity="secondary" size="small" @click="q = ''; language = ''; sort = 'viewers'; dir = 'desc'" />
      </div>
    </Dialog>
  </section>
</template>

<style scoped>
.moment-bar { display: flex; flex-wrap: wrap; align-items: center; gap: 0.5rem; margin-bottom: 0.5rem; }
.moment-timestamp { font-size: 0.85rem; white-space: nowrap; }
.moment-spacer { flex: 1; }
.moment-controls { display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap; }
.moment-nav { display: flex; gap: 0.25rem; flex-shrink: 0; }

.sr-only { position: absolute; width: 1px; height: 1px; padding: 0; margin: -1px; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; border: 0; }

.now-btn { position: relative; }
.now-glow { box-shadow: 0 0 8px 2px var(--p-primary-color, #6366f1); }

.retry-btn { background: none; border: 1px solid var(--p-primary-color, #6366f1); border-radius: 3px; color: var(--p-primary-color, #6366f1); cursor: pointer; font-size: 0.8rem; padding: 0.15rem 0.4rem; vertical-align: middle; }
.retry-btn:hover { background: var(--p-primary-color, #6366f1); color: #fff; }

.retention-warn {
  background: rgba(220, 38, 38, 0.12); border: 1px solid rgba(220, 38, 38, 0.3);
  border-radius: 4px; padding: 0.4rem 0.75rem; font-size: 0.8rem; color: #fca5a5; margin: 0 0 0.5rem;
}

.survivor-search { position: relative; display: flex; align-items: center; gap: 0.4rem; flex: 1; max-width: 380px; min-width: 180px; }
.survivor-input {
  width: 100%; padding: 0.35rem 2rem 0.35rem 1.8rem; font-size: 0.85rem;
  border: 1px solid var(--p-inputtext-border-color); border-radius: 4px;
  background: var(--p-inputtext-background); color: var(--p-inputtext-color);
}
.search-icon { position: absolute; left: 0.55rem; font-size: 0.8rem; opacity: 0.6; pointer-events: none; }
.sort-hint {
  position: absolute; right: 1.6rem; font-size: 0.65rem; text-transform: uppercase;
  letter-spacing: 0.03em; color: var(--p-primary-color); white-space: nowrap; pointer-events: none;
}
.input-wrap { position: relative; display: flex; align-items: center; }
.input-clear { position: absolute; right: 0.5rem; font-size: 0.75rem; opacity: 0.5; cursor: pointer; background: none; border: none; color: inherit; padding: 0; }
.input-clear:hover { opacity: 0.9; }

@media (max-width: 640px) {
  .moment-timestamp { font-size: 0.75rem; }
  .filter-btn :deep(.p-button-label) { display: none; }
  .moment-controls { flex-direction: column; align-items: stretch; }
  .survivor-search { max-width: 100%; }
}

.field-hint { font-size: 0.72rem; margin: 0; }
.filters-grid { display: flex; flex-direction: column; gap: 0.75rem; }
.filters-grid .field { display: flex; flex-direction: column; gap: 0.2rem; }
.filters-grid label { font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.04em; color: var(--p-text-muted-color); }

.gallery-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(360px, 1fr)); gap: 0.75rem; margin-bottom: 0.5rem; }
@media (max-width: 640px) { .gallery-grid { grid-template-columns: 1fr; } }

.gallery-item { display: flex; flex-direction: column; gap: 0.25rem; content-visibility: auto; contain-intrinsic-size: auto 360px 100px; }
.gallery-headline { display: flex; align-items: center; justify-content: space-between; gap: 0.4rem; }
.gallery-name { font-size: 0.8rem; font-weight: 600; color: inherit; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; text-decoration: none; }
.gallery-thumb-link { display: block; }
.gallery-thumb { width: 100%; aspect-ratio: 16/9; object-fit: cover; max-width: 1280px; border-radius: 4px; background: #000; display: block; }

.score-badge { flex: 0 0 auto; font-size: 0.65rem; font-weight: 700; padding: 0.05rem 0.35rem; border-radius: 3px; color: #fff; }

.loading-spinner { display: flex; justify-content: center; padding: 2rem 0; }
.scroll-sentinel { display: flex; justify-content: center; padding: 1rem 0; min-height: 40px; }
.spinner { width: 32px; height: 32px; border: 3px solid var(--p-surface-700); border-top-color: var(--p-primary-color); border-radius: 50%; animation: spin 0.7s linear infinite; }
.small-spinner { width: 20px; height: 20px; border-width: 2px; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
